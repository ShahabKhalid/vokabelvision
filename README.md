# VokabelVision

**VokabelVision** is an open-source project designed to help learners master German vocabulary through engaging Instagram Reels. This project automates content creation and publishing by combining multiple APIs to generate words, visuals, audio, and videos for Instagram.

## Features

- **Automated Vocabulary Generation:**  
  Uses the OpenAI ChatGPT API to generate random German vocabulary words (with articles when possible), their English translations, a creative reel caption including relevant hashtags for German learning, and concise example sentences (each not exceeding 10 words).

- **Visual Creation with Leonardo.ai:**  
  Generates engaging visuals based on prompts designed for vocabulary learning using Leonardo.ai. The visuals are tailored for Instagram, ensuring your posts are both informative and visually appealing.

- **Audio Generation with ElevenLabs:**  
  Produces high-quality German pronunciation audio (with options for SSML-based adjustments like pauses and slow speech) using ElevenLabs’ text-to-speech API.

- **Instagram Publishing:**  
  Publishes content to Instagram Reels via the Instagram Graph API. The system automatically uploads video content that combines the generated visual and audio.

- **Cloudinary Integration for Video Hosting:**  
  Uses Cloudinary’s API (via the cloudinary-go package) to upload generated video files. The publicly accessible video URL is then passed to Instagram for publishing. Files can be deleted after publishing to save resources.

- **Duplication Prevention:**  
  Maintains a list of the last 50 posted vocabulary words (stored in a JSON file) and includes this exclusion list in the ChatGPT prompt to avoid repeating content.

- **Configurable and Modular:**  
  All configuration (API keys, credentials, etc.) is managed through a `config/config.json` file. A sample configuration is provided as `config/config.json.example` in the repository.

- **Scheduling & CLI:**  
  Supports both one-time and scheduled publishing. For example, you can run the app once using a CLI flag (`--once`) or schedule posts at specific times (e.g., 07:00, 13:00, and 19:00 Berlin time) using a cron job.

## Technology Stack

- **Go (Golang):**  
  The core application logic is written in Go for performance and simplicity.

- **OpenAI ChatGPT API:**  
  Generates dynamic vocabulary content and ensures freshness by excluding recently posted words.

- **Leonardo.ai API:**  
  Creates custom visuals for each vocabulary word tailored for Instagram posts.

- **ElevenLabs API:**  
  Converts text into natural-sounding German audio for pronunciation practice.

- **Instagram Graph API:**  
  Publishes video content (formatted as Reels) to your Instagram Business/Creator account.

- **Cloudinary Go SDK:**  
  Uploads and manages video files, providing a publicly accessible URL for Instagram.

- **robfig/cron:**  
  Schedules the automatic posting of content at specified times.

## Getting Started

### Prerequisites

- **Go 1.16+** installed.
- **API Credentials:**  
  - OpenAI API Key for ChatGPT.
  - Leonardo.ai API credentials.
  - ElevenLabs API Key.
  - Instagram Graph API bearer token (for a Business/Creator account).
  - Cloudinary account credentials.
- **Configuration:**  
  Copy `config/config.json.example` to `config/config.json` and fill in your API keys, tokens, and other settings.

### Installation

1. **Clone the Repository:**
   ```bash
   git clone https://github.com/yourusername/vokabelvision.git
   cd vokabelvision
   ```

2. **Configure Environment:**  
   Update the `config/config.json` file with your credentials. See the provided example in `config/config.json.example`.

3. **Install Dependencies:**
   ```bash
   go mod download
   ```

4. **Build the Project:**
   ```bash
   go build
   ```

### Running the Application

- **One-Time Execution:**
  Run the CLI with the `--once` flag to generate and publish content immediately:
  ```bash
  go run main.go --once
  ```

- **Scheduled Execution:**
  Without the `--once` flag, the application schedules posts (for example, 07:00, 13:00, and 19:00 Berlin time).

## Folder Structure

- `main.go`  
  The main entry point which sets up scheduling and CLI options.

- `config/`  
  Contains configuration files. Rename `config.json.example` to `config.json` and update with your credentials.

- `chatgpt/`  
  Contains logic for interacting with ChatGPT to generate vocabulary, captions, and sample sentences.

- `instagram/`  
  Contains functions to publish videos to Instagram via the Graph API.

- `cloudinary/`  
  Contains functions to upload and manage videos using the Cloudinary Go SDK.

- `posted_vocabs.json`  
  A JSON file storing the last 50 vocabulary words to prevent duplicates.

## Contributing

Contributions are welcome! Please submit issues or pull requests for improvements, bug fixes, or new features. Ensure you follow the project's code style and document your changes.

## License

This project is open-sourced under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [OpenAI ChatGPT API](https://openai.com/api/)
- [Leonardo.ai](https://www.leonardo.ai/)
- [ElevenLabs](https://www.elevenlabs.io/)
- [Instagram Graph API](https://developers.facebook.com/docs/instagram-api/)
- [Cloudinary](https://cloudinary.com/)
- [robfig/cron](https://github.com/robfig/cron)
