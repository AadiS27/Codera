FROM eclipse-temurin:17-jdk-jammy

# Create a non-root user 'executor'
RUN useradd -m -d /sandbox -s /bin/bash executor

# Create workspace directory
RUN mkdir -p /workspace && chown executor:executor /workspace

# Switch to the non-root user
USER executor
WORKDIR /workspace

# We will run the container with 'sleep infinity' and use 'docker exec' to run commands
CMD ["sleep", "infinity"]
