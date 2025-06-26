# Tool Content Backend

A Go-based backend service for content processing with AI capabilities.

## Features

- Text-to-Speech processing
- Voice separation and processing
- AI-powered content generation
- User authentication and management
- File upload and processing

## Prerequisites

- Go 1.19 or higher
- PostgreSQL database
- OpenAI API key
- Google Cloud credentials for Text-to-Speech

## Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd tool_content_be
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up environment variables**
   ```bash
   cp env.example .env
   # Edit .env with your actual credentials
   ```

4. **Set up credentials**
   - Create a `data/` directory
   - Place your Google Cloud service account JSON file in `data/google_cloud_tts_api.json`
   - Update the `.env` file with your OpenAI API key and other credentials

5. **Set up pretrained models**
   - Create a `pretrained_models/` directory
   - Download the required model files (see Model Setup section)

6. **Run the application**
   ```bash
   go run main.go
   ```

## Model Setup

The application requires pretrained models for voice separation. These files are large (>50MB) and are not included in the repository.

### Required Models

Place the following files in `pretrained_models/2stems/`:
- `model.data-00000-of-00001` (75MB)
- `model.index` (5KB)
- `model.meta` (787KB)
- `checkpoint` (67B)

### Download Instructions

1. Download the HTDemucs 2-stems model from the official repository
2. Extract the files to `pretrained_models/2stems/`
3. Ensure all files are present and have the correct names

## Environment Variables

See `env.example` for all required environment variables.

### Required Credentials

- **OpenAI API Key**: For AI content generation
- **Google Cloud Service Account**: For Text-to-Speech functionality
- **Database Credentials**: PostgreSQL connection details
- **JWT Secret**: For user authentication

## File Structure

```
tool_content_be/
├── config/          # Configuration files
├── handler/         # HTTP handlers
├── service/         # Business logic services
├── middleware/      # HTTP middleware
├── router/          # Route definitions
├── storage/         # File storage (gitignored)
├── data/           # Credentials and data (gitignored)
├── pretrained_models/ # AI models (gitignored)
└── util/           # Utility functions
```

## Security Notes

- Never commit `.env` files or credential files
- The `data/`, `storage/`, and `pretrained_models/` directories are gitignored
- Use environment variables for all sensitive configuration
- Keep your API keys secure and rotate them regularly

## Docker

To run with Docker:

```bash
docker-compose up -d
```

## API Documentation

The API endpoints are defined in the `router/` directory. Main endpoints include:

- `/api/auth/*` - Authentication endpoints
- `/api/upload` - File upload
- `/api/process` - Content processing
- `/api/tts` - Text-to-Speech
- `/api/history` - Processing history

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Ensure all tests pass
5. Submit a pull request

## License

[Add your license information here] 