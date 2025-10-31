#!/bin/bash
set -e

echo "🔨 Rebuilding listmonk with campaign analytics enhancements..."

# Build frontend
echo "📦 Building frontend..."
cd /home/adam/listmonk/frontend
yarn build

# Build backend
echo "🔧 Building backend..."
cd /home/adam/listmonk
CGO_ENABLED=0 go build -o listmonk cmd/*.go

echo "✅ Build complete!"
echo ""
echo "To start listmonk, run:"
echo "  cd /home/adam/listmonk"
echo "  ./listmonk"
echo ""
echo "Or if running in Docker:"
echo "  docker-compose restart"
