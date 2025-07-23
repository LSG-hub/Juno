# Flutter Web Dockerfile for containerized deployment
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    unzip \
    xz-utils \
    zip \
    libglu1-mesa \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user
RUN useradd -m -u 1000 developer
USER developer
WORKDIR /home/developer

# Install Flutter
ENV FLUTTER_HOME=/home/developer/flutter
ENV PATH="$FLUTTER_HOME/bin:$PATH"

RUN git clone https://github.com/flutter/flutter.git -b stable --depth 1 $FLUTTER_HOME
RUN flutter doctor
RUN flutter config --enable-web

# Set working directory for app
WORKDIR /home/developer/app

# Copy pubspec file first for dependency caching
COPY --chown=developer:developer pubspec.yaml ./

# Get dependencies
RUN flutter pub get

# Copy the rest of the app
COPY --chown=developer:developer . .

# Build for web with environment variables
ARG COORDINATOR_HOST=localhost
ARG COORDINATOR_PORT=8091
RUN flutter build web --release --dart-define=COORDINATOR_HOST=$COORDINATOR_HOST --dart-define=COORDINATOR_PORT=$COORDINATOR_PORT

# Use nginx to serve the built app
FROM nginx:alpine

# Copy built web app to nginx
COPY --from=0 /home/developer/app/build/web /usr/share/nginx/html

# Copy custom nginx config for Flutter web app
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port 80
EXPOSE 80

# Start nginx
CMD ["nginx", "-g", "daemon off;"]