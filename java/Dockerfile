FROM eclipse-temurin:21-jdk-alpine AS build
WORKDIR /src
COPY java/src ./src
RUN mkdir -p /out && javac -d /out src/RateLimiter.java src/App.java

FROM eclipse-temurin:21-jre-alpine
COPY --from=build /out /app
ENTRYPOINT ["java", "-cp", "/app", "demo.App"]
