package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type endpoint struct {
	Name    string
	Path    string
	Payload func() any
}

type jobResponse struct {
	Success bool   `json:"success"`
	JobID   string `json:"job_id"`
	Error   string `json:"error"`
}

type jobStatus struct {
	Job struct {
		Status string `json:"status"`
	} `json:"job"`
}

func main() {
	baseURL := flag.String("url", "http://localhost:8080", "Backend base URL")
	total := flag.Int("n", 50, "Total requests to send")
	concurrency := flag.Int("c", 5, "Concurrent workers")
	pollJobs := flag.Bool("poll", true, "Poll jobs until completion")
	pollInterval := flag.Duration("poll-interval", 2*time.Second, "Poll interval for job status")
	pollTimeout := flag.Duration("poll-timeout", 5*time.Minute, "Max time to wait for a job")
	numUsers := flag.Int("users", 5, "Number of simulated users")
	errorRate := flag.Float64("error-rate", 0.2, "Fraction of requests that send bad payloads (0.0-1.0)")
	flag.Parse()

	// Generate user pool
	users := make([]string, *numUsers)
	log.Println("=== Simulated Users ===")
	for i := range users {
		users[i] = uuid.Must(uuid.NewV7()).String()
		log.Printf("  User %d: %s", i+1, users[i])
	}
	log.Println()

	endpoints := []endpoint{
		{
			Name: "nanobanana/generate",
			Path: "/api/v1/nanobanana/generate",
			Payload: func() any {
				return map[string]any{
					"prompt":     randomPrompt(imagePrompts),
					"image_size": randomChoice([]string{"square", "square_hd", "portrait_4_3", "landscape_16_9"}),
					"num_images": 1,
				}
			},
		},
		{
			Name: "sounds/tts",
			Path: "/api/v1/sounds/tts",
			Payload: func() any {
				return map[string]any{
					"text":     randomPrompt(ttsTexts),
					"voice_id": randomChoice([]string{"EXAVITQu4vr4xnSDxMaL", "21m00Tcm4TlvDq8ikWAM", "AZnzlk1XvdvUeBnXmlld"}),
				}
			},
		},
		{
			Name: "sounds/sound-effects",
			Path: "/api/v1/sounds/sound-effects",
			Payload: func() any {
				return map[string]any{
					"text":             randomPrompt(sfxPrompts),
					"duration_seconds": 3.0 + rand.Float64()*7.0,
				}
			},
		},
		{
			Name: "sounds/music",
			Path: "/api/v1/sounds/music",
			Payload: func() any {
				return map[string]any{
					"text":             randomPrompt(musicPrompts),
					"duration_seconds": 5.0 + rand.Float64()*15.0,
				}
			},
		},
	}

	// Bad payloads that trigger 400 errors (missing required fields)
	errorPayloads := []endpoint{
		{
			Name: "nanobanana/generate (bad)",
			Path: "/api/v1/nanobanana/generate",
			Payload: func() any {
				return map[string]any{}
			},
		},
		{
			Name: "sounds/tts (bad)",
			Path: "/api/v1/sounds/tts",
			Payload: func() any {
				return map[string]any{"text": ""}
			},
		},
		{
			Name: "sounds/sound-effects (bad)",
			Path: "/api/v1/sounds/sound-effects",
			Payload: func() any {
				return map[string]any{}
			},
		},
		{
			Name: "sounds/music (bad)",
			Path: "/api/v1/sounds/music",
			Payload: func() any {
				return map[string]any{}
			},
		},
		{
			Name: "jobs/status (not found)",
			Path: "/api/v1/jobs/nonexistent-job-id/status",
			Payload: nil, // GET request
		},
	}

	client := &http.Client{Timeout: 30 * time.Second}

	var (
		sent       int64
		succeeded  int64
		failed     int64
		errorsSent int64
		completed  int64
		jobsFailed int64
	)

	type jobTrack struct {
		endpoint string
		jobID    string
	}

	jobsCh := make(chan jobTrack, *total)
	workCh := make(chan int, *total)

	// Fill work channel
	for i := 0; i < *total; i++ {
		workCh <- i
	}
	close(workCh)

	log.Println("=== Stress Test ===")
	log.Printf("Target:      %s", *baseURL)
	log.Printf("Requests:    %d", *total)
	log.Printf("Concurrency: %d", *concurrency)
	log.Printf("Users:       %d", *numUsers)
	log.Printf("Error rate:  %.0f%%", *errorRate*100)
	log.Printf("Poll jobs:   %v", *pollJobs)
	log.Println()

	start := time.Now()

	// Submit phase
	var wg sync.WaitGroup
	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range workCh {
				userID := users[rand.Intn(len(users))]
				atomic.AddInt64(&sent, 1)

				// Decide if this request should trigger an error
				isError := rand.Float64() < *errorRate

				var ep endpoint
				if isError {
					ep = errorPayloads[rand.Intn(len(errorPayloads))]
					atomic.AddInt64(&errorsSent, 1)
				} else {
					ep = endpoints[rand.Intn(len(endpoints))]
				}

				// Handle GET-based error endpoints (e.g., job not found)
				var resp *http.Response
				var err error
				if ep.Payload == nil {
					req, _ := http.NewRequest(http.MethodGet, *baseURL+ep.Path, nil)
					req.Header.Set("X-User-ID", userID)
					resp, err = client.Do(req)
				} else {
					body, _ := json.Marshal(ep.Payload())
					req, _ := http.NewRequest(http.MethodPost, *baseURL+ep.Path, bytes.NewReader(body))
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("X-User-ID", userID)
					resp, err = client.Do(req)
				}

				if err != nil {
					atomic.AddInt64(&failed, 1)
					log.Printf("[FAIL] %s user=%s: %v", ep.Name, userID[:8], err)
					continue
				}

				respBody, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				if isError {
					log.Printf("[ERROR_REQ] %s user=%s status=%d", ep.Name, userID[:8], resp.StatusCode)
					continue
				}

				if resp.StatusCode != http.StatusAccepted {
					atomic.AddInt64(&failed, 1)
					log.Printf("[FAIL] %s user=%s status=%d body=%s", ep.Name, userID[:8], resp.StatusCode, string(respBody))
					continue
				}

				var jr jobResponse
				if err := json.Unmarshal(respBody, &jr); err != nil || !jr.Success {
					atomic.AddInt64(&failed, 1)
					log.Printf("[FAIL] %s user=%s bad response: %s", ep.Name, userID[:8], string(respBody))
					continue
				}

				atomic.AddInt64(&succeeded, 1)
				if *pollJobs {
					jobsCh <- jobTrack{endpoint: ep.Name, jobID: jr.JobID}
				}
			}
		}()
	}
	wg.Wait()
	close(jobsCh)

	submitDuration := time.Since(start)
	log.Printf("\n=== Submit Phase Done (%.1fs) ===", submitDuration.Seconds())
	log.Printf("Sent: %d | Accepted: %d | Errors triggered: %d | Failed: %d", sent, succeeded, errorsSent, failed)

	if !*pollJobs {
		printSummary(sent, succeeded, failed, errorsSent, 0, 0, time.Since(start))
		return
	}

	// Poll phase
	log.Println("\n=== Polling Jobs ===")
	var pollWg sync.WaitGroup
	pollClient := &http.Client{Timeout: 10 * time.Second}

	// Use concurrency for polling too
	pollWork := make(chan jobTrack, len(jobsCh))
	for jt := range jobsCh {
		pollWork <- jt
	}
	close(pollWork)

	for w := 0; w < *concurrency; w++ {
		pollWg.Add(1)
		go func() {
			defer pollWg.Done()
			for jt := range pollWork {
				deadline := time.Now().Add(*pollTimeout)
				for time.Now().Before(deadline) {
					resp, err := pollClient.Get(fmt.Sprintf("%s/api/v1/jobs/%s/status", *baseURL, jt.jobID))
					if err != nil {
						time.Sleep(*pollInterval)
						continue
					}
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()

					var js jobStatus
					json.Unmarshal(body, &js)

					switch js.Job.Status {
					case "completed":
						atomic.AddInt64(&completed, 1)
						log.Printf("[DONE] %s job=%s", jt.endpoint, jt.jobID)
						goto next
					case "failed":
						atomic.AddInt64(&jobsFailed, 1)
						log.Printf("[JOB_FAIL] %s job=%s", jt.endpoint, jt.jobID)
						goto next
					}
					time.Sleep(*pollInterval)
				}
				atomic.AddInt64(&jobsFailed, 1)
				log.Printf("[TIMEOUT] %s job=%s", jt.endpoint, jt.jobID)
			next:
			}
		}()
	}
	pollWg.Wait()

	printSummary(sent, succeeded, failed, errorsSent, completed, jobsFailed, time.Since(start))
}

func printSummary(sent, succeeded, failed, errorsSent, completed, jobsFailed int64, elapsed time.Duration) {
	log.Println("\n=== Summary ===")
	log.Printf("Duration:         %.1fs", elapsed.Seconds())
	log.Printf("Requests sent:    %d", sent)
	log.Printf("Accepted:         %d", succeeded)
	log.Printf("Errors triggered: %d", errorsSent)
	log.Printf("Submit errors:    %d", failed)
	if completed+jobsFailed > 0 {
		log.Printf("Jobs completed:   %d", completed)
		log.Printf("Jobs failed:      %d", jobsFailed)
	}
	log.Printf("Throughput:       %.1f req/s", float64(sent)/elapsed.Seconds())
}

func randomChoice(items []string) string {
	return items[rand.Intn(len(items))]
}

func randomPrompt(prompts []string) string {
	return prompts[rand.Intn(len(prompts))]
}

var imagePrompts = []string{
	"A serene lake surrounded by mountains at sunset",
	"A busy street market in a Latin American town",
	"A tropical fish jumping out of crystal clear water",
	"A rustic wooden cabin next to a fishing pond",
	"A family enjoying outdoor dining at a countryside restaurant",
	"Fresh grilled tilapia on a wooden plate with salad",
	"A colorful hummingbird feeding on tropical flowers",
	"Aerial view of a campestre restaurant with fishing ponds",
	"A child catching a fish with a big smile",
	"Traditional Colombian countryside landscape with green hills",
}

var ttsTexts = []string{
	"Bienvenidos a Laguna Escondida, el mejor destino de pesca deportiva en Antioquia.",
	"Disfruta de la mejor experiencia gastronómica campestre con tu familia.",
	"Tenemos seis especies de peces diferentes para la pesca deportiva.",
	"Reserva tu mesa hoy y vive una experiencia única en la naturaleza.",
	"Welcome to the best sport fishing experience in Colombia.",
	"Our restaurant offers fresh fish prepared in traditional Colombian style.",
	"Come and enjoy a day of fishing and great food with your family.",
	"La Primavera, Marinilla, te espera con la mejor pesca y gastronomía.",
}

var sfxPrompts = []string{
	"Water splashing as a fish jumps",
	"Birds chirping in a tropical forest",
	"Sizzling sound of fish frying on a grill",
	"Children laughing and playing outdoors",
	"A fishing reel spinning fast",
	"Gentle stream flowing over rocks",
	"Rain falling on a tin roof",
	"Crickets and frogs at night in the countryside",
}

var musicPrompts = []string{
	"Upbeat tropical acoustic guitar background music",
	"Relaxing ambient nature sounds with soft piano",
	"Energetic Latin pop for restaurant promo video",
	"Calm acoustic music for a fishing adventure video",
	"Happy folk music for a family outing commercial",
	"Chill lo-fi beats for a travel vlog background",
}
