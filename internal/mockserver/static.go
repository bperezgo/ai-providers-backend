package mockserver

import "net/http"

// RegisterStaticRoutes serves tiny valid media files for provider download URLs.
func RegisterStaticRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /mock-files/sample.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(samplePNG())
	})

	mux.HandleFunc("GET /mock-files/sample.mp3", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(sampleMP3())
	})

	mux.HandleFunc("GET /mock-files/sample.mp4", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.Write(sampleMP4())
	})
}

// samplePNG returns a minimal valid 1x1 PNG (67 bytes).
func samplePNG() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, // 8-bit RGB
		0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, 0x54, // IDAT chunk
		0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00, 0x00, // compressed data
		0x00, 0x02, 0x00, 0x01, 0xE2, 0x21, 0xBC, 0x33, // CRC
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, // IEND chunk
		0xAE, 0x42, 0x60, 0x82, // IEND CRC
	}
}

// sampleMP3 returns a minimal valid MP3 frame (~48 bytes).
// MPEG1 Layer3, 128kbps, 44100Hz, stereo, padded.
func sampleMP3() []byte {
	// Valid MP3 frame header: 0xFF 0xFB 0x90 0x00
	// Followed by zero-padded frame data
	frame := make([]byte, 417) // standard frame size for 128kbps/44100Hz
	frame[0] = 0xFF
	frame[1] = 0xFB
	frame[2] = 0x90
	frame[3] = 0x00
	return frame
}

// sampleMP4 returns a minimal valid MP4 container (~40 bytes).
func sampleMP4() []byte {
	// ftyp box: file type
	ftyp := []byte{
		0x00, 0x00, 0x00, 0x14, // box size = 20
		0x66, 0x74, 0x79, 0x70, // "ftyp"
		0x69, 0x73, 0x6F, 0x6D, // major brand "isom"
		0x00, 0x00, 0x02, 0x00, // minor version
		0x69, 0x73, 0x6F, 0x6D, // compatible brand "isom"
	}
	// moov box: movie (empty)
	moov := []byte{
		0x00, 0x00, 0x00, 0x08, // box size = 8
		0x6D, 0x6F, 0x6F, 0x76, // "moov"
	}
	return append(ftyp, moov...)
}
