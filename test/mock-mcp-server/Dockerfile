FROM python:3.11-slim

WORKDIR /app

# Copy the server script
COPY main.py .

# Make it executable
RUN chmod +x main.py

# Expose port
EXPOSE 8080

# Run the server
CMD ["python", "main.py"]
