# Adaptive Bitrate

A simple, configurable adaptive bitrate (ABR) project for streaming media. This repository provides utilities and example code for selecting video quality dynamically based on measured network conditions and player buffer state.

## Features
- Basic ABR algorithms (bandwidth-based, buffer-based)
- Configurable thresholds and smoothing
- Example integration hooks for players and streaming clients

## Prerequisites
- Node.js 14+ (or adjust for your runtime)
- npm or yarn

## Installation
1. Clone the repo:
   git clone <repo-url>
2. Install dependencies:
   npm install

(If this is not a git repo, just copy the files into your workspace.)

## Usage
- Configure policy in config/ or via environment variables.
- Run the demo server (if provided):
  npm start

Integration example (pseudo):
```js
const abr = require('./lib/abr');
const policy = new abr.Policy({ strategy: 'bandwidth' });
player.on('progress', () => {
  const quality = policy.selectQuality(player.getBuffer(), player.getMeasuredBandwidth());
  player.setQuality(quality);
});
```

## Configuration
- strategy: 'bandwidth' | 'buffer' | 'hybrid'
- minBitrate / maxBitrate: range of supported bitrates
- smoothingWindow: milliseconds for bandwidth smoothing

## Development
- Follow standard Node.js project flow.
- Run linter/tests if present:
  npm test

## Contributing
Issues and pull requests are welcome. Please include tests and update documentation for new features.

## Docker

A multi-stage Dockerfile is provided to build the Go binary and produce a small runtime image.

Build the image (uses Go 1.20):

```bash
docker build -t adaptive-bitrate:latest .
```

Run the container (exposes port 8080 by default):

```bash
docker run -p 8080:8080 --rm adaptive-bitrate:latest
```

Notes:
- The Dockerfile runs `go build -o /app/app` in the builder stage. If your main package or output path differs, update the Dockerfile.
- To pass runtime configuration, use environment variables: `-e PORT=8080` or bind mount a config file.

## License
Specify your license here (e.g., MIT). Replace this line with the chosen license.

## Contact
For questions, open an issue or contact the maintainer.
