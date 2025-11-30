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



## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.


## License

See [LICENSE.md](LICENSE.md)
