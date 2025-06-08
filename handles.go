package main

import (
    "bytes"
    "encoding/json"
    "io/ioutil"
    "net/http"
    "time"
    "context"
    "fmt"
    "log"
    "os"
    "os/exec"
    "google.golang.org/genai"
    "regexp"
    "github.com/stianeikeland/go-rpio"
)

var (
        pin = rpio.Pin(17)
	redPin = rpio.Pin(14)
	greenPin = rpio.Pin(16)
)

const (
        assemblyApiKey = "YOUR KEY HERE"
        geminiApiKey = "YOUR KEY HERE"

)

func main() {

	setupPins()

	alwaysListening()

}

func setupPins() {

	 if err := rpio.Open(); err != nil {// Open and map memory to access gpi>

                fmt.Println(err)
                os.Exit(1)
        }

        greenPin.Output()
        greenPin.High()
        redPin.Output()
        redPin.High()
	pin.Output()       // Output mode
        pin.High()         // Set pin High
        pin.Toggle()	   // toggle it to turn it off/on
        redPin.Toggle()
	greenPin.Toggle()
}

func askHandles(input string) {
    
	nig := fmt.Sprintf("Respond as a cyberman from doctor who but still provide real world facts, while remaining short and concise, don't use special characters: %s", input)
    ctx := context.Background()
    client, err := genai.NewClient(ctx, &genai.ClientConfig{
        APIKey:  geminiApiKey,
        Backend: genai.BackendGeminiAPI,
    })
    if err != nil {
        log.Fatal(err)
    }
    //Feed prompt to gemini api here
    result, err := client.Models.GenerateContent(
        ctx,
        "gemini-2.0-flash",
	genai.Text(nig),
        nil,
    )
    if err != nil {
        log.Fatal(err)
    }

    //redPin.Toggle()
    fmt.Println("Handles output: ", result.Text())
    pin.Toggle()
    espeak(result.Text())
    pin.Toggle()
}

func alwaysListening() {
	//listening for output
	loop: for {
		greenPin.Toggle()
		sox("5")
		greenPin.Toggle()
		redPin.Toggle()
		uploadUrl := upload("output.wav")
	        fmt.Println("upload url: ", uploadUrl)
        	id := transcribe(uploadUrl)
        	fmt.Println("transcript id: ", id)
        	result1 := poll(id)
        	//delete audio file after it's been processed 
        	deleteFile("output.wav")
		redPin.Toggle()
		fmt.Println("user input: ", result1)
		if containsWord(result1, "Handles") || containsWord(result1, "handles") {
			//pin.Toggle()
			//espeak("Hello there")
			//pin.Toggle()
			askHandles(result1)
		
		}
		if containsWord(result1, "Goodbye") || containsWord(result1, "goodbye") {
                        pin.Toggle()
			espeak("Goodbye")
			pin.Toggle()
			defer rpio.Close()// Unmap gpio memory when done
                        break loop

                }

	}
}

func containsWord(text, word string) bool {
	// \b matches word boundaries
	pattern := `\b` + regexp.QuoteMeta(word) + `\b`
	matched, err := regexp.MatchString(pattern, text)
	if err != nil {
		return false
	}
	return matched
}

//string must be num like 5
func sox(num string) {

        // Output file name
        outputFile := "output.wav"

        // Use `rec` to record for 5 seconds
        cmd := exec.Command("sox", "-d", outputFile, "trim", "0", num)

        // Start recording
        log.Println("Recording for 5 seconds...")
        err := cmd.Run()
        if err != nil {
                log.Fatalf("Failed to record audio: %v", err)
        }

        //log.Println("Recording done, file saved as", outputFile)

}

func deleteFile(filename string) {
	err := os.Remove(filename)
	if err != nil {
		log.Printf("Failed to delete file %s: %v", filename, err)
	} else {
		log.Printf("Deleted file: %s", filename)
	}
}

func espeak(word string) {

        cmd := exec.Command("espeak", word)
        cmd.Stdout = os.Stdout
        cmd.Run()
}

//you can tell I stole this part
func poll(id string) string {
	const API_KEY = assemblyApiKey
	const TRANSCRIPT_URL = "https://api.assemblyai.com/v2/transcript"

	client := &http.Client{}

	for {
		req, _ := http.NewRequest("GET", TRANSCRIPT_URL+"/"+id, nil)
		req.Header.Set("content-type", "application/json")
		req.Header.Set("authorization", API_KEY)

		res, err := client.Do(req)
		if err != nil {
			log.Fatalln(err)
		}
		defer res.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(res.Body).Decode(&result)

		fmt.Printf("Polling status: %+v\n", result["status"])

		status := result["status"].(string)

		if status == "completed" {
			fmt.Println("Transcription completed.")
			return result["text"].(string)
		} else if status == "error" {
			fmt.Println("Transcription error:", result["error"])
			return "error"
		}

		time.Sleep(3 * time.Second) // Wait before polling again
	}
}

func transcribe(uploadurl string) string {
        //audio url here from upload script
	AUDIO_URL := uploadurl
        const API_KEY = assemblyApiKey
        const TRANSCRIPT_URL = "https://api.assemblyai.com/v2/transcript"

        // prepare json data
        values := map[string]string{"audio_url": AUDIO_URL}
        jsonData, _ := json.Marshal(values)

        // setup HTTP client and set header
        client := &http.Client{}
        req, _ := http.NewRequest("POST", TRANSCRIPT_URL, bytes.NewBuffer(jsonData))
        req.Header.Set("content-type", "application/json")
        req.Header.Set("authorization", API_KEY)
        res, _ := client.Do(req)

        defer res.Body.Close()

        // decode json and store it in a map
        var result map[string]interface{}
        json.NewDecoder(res.Body).Decode(&result)

        // print the id of the transcribed audio
        //fmt.Println(result["id"])
	return result["id"].(string)
}

func upload(filename string) string {
        const API_KEY = assemblyApiKey
        const UPLOAD_URL = "https://api.assemblyai.com/v2/upload"
	//uplaod url is there for the mfing api
        // Load file
        data, err := ioutil.ReadFile(filename)
        if err != nil {
                log.Fatalln(err)
        }
        //fmt.Println("here")
        // Setup HTTP client and set header
        client := &http.Client{}
        req, _ := http.NewRequest("POST", UPLOAD_URL, bytes.NewBuffer(data))
        req.Header.Set("authorization", API_KEY)
        res, err := client.Do(req)

        if err != nil {
                log.Fatalln(err)
        }

        // decode json and store it in a map
        var result map[string]interface{}
        json.NewDecoder(res.Body).Decode(&result)

        // return the upload_url
        //fmt.Println(result["upload_url"])
	return result["upload_url"].(string)
}
