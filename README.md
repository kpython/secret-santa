# 🎁 Secret Santa Web App

A simple, privacy-focused Secret Santa organizer

## The story

This little app was born out of a real family struggle. Every year, my sister tries to organize a Secret Santa for our family, but last time she faced a nightmare: a bunch of apps that required installing software, creating accounts, in-app purchases, or even tracking your activities.

So, I decided to build a simple tool for our family: no accounts, no ads, no nonsense. Just a clean, easy-to-use way for everyone to join the fun and enjoy the Secret Santa spirit.


## Quick Start

1. **Install Go** (1.16 or higher)

2. **Clone and run**
   ```bash
   git clone <your-repo-url>
   cd secret-santa
   go run main.go
   ```

3. **Access the app**
   Open http://localhost:8080


## Deploy to Render

### One-Click Deploy

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy)

### Manual Deploy

1. Fork this repository
2. Create a new Web Service on [Render](https://render.com)
3. Connect your GitHub repository
4. Render will automatically detect the `render.yaml` and configure:
   - Docker build from `Dockerfile`
   - Persistent disk for data storage (1GB)
   - Environment variables

The app will be available at `https://your-app-name.onrender.com`

**Note:** The free tier on Render will spin down after 15 minutes of inactivity. The first request after inactivity may take 30-60 seconds.


## Run with Docker

### Build and run locally

```bash
# Build the image
docker build -t secret-santa .

# Run the container
docker run -p 8080:8080 -v $(pwd)/data:/app secret-santa
```

Access the app at http://localhost:8080

### Using Docker Compose (optional)

```bash
docker-compose up
```


## Features

- 🔒 **Privacy-focused** - No accounts, no tracking, no data collection
- 🎲 **Cryptographically secure** - Uses 32-character tokens (2^128 entropy)
- 🌍 **Multi-language** - Supports English, French, German, and Portuguese
- 📱 **Mobile responsive** - Works on all devices
- ⚡ **Lightweight** - Simple Go application with minimal dependencies
- 🗑️ **Auto-cleanup** - Draws older than 30 days are automatically deleted


## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.


## License

See LICENSE.md
