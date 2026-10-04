FROM eclipse-temurin:21-jdk-alpine AS build
WORKDIR /src
COPY clients/java/src/main/java/ratelimiter/RateLimiter.java /src/ratelimiter/RateLimiter.java
COPY demo/java-backend/src/main/java/demo/Main.java /src/demo/Main.java
RUN mkdir -p /out && javac -d /out /src/ratelimiter/RateLimiter.java /src/demo/Main.java

FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
COPY --from=build /out /app/out
CMD ["java", "-cp", "/app/out", "demo.Main"]
