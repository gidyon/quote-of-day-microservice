FROM scratch

WORKDIR /

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY quote-app /

# This container exposes port 8080 to the outside world
EXPOSE 8080

# Run the executable
CMD ["/quote-app"]