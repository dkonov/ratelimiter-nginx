FROM eclipse-temurin:21-jdk-alpine AS build
WORKDIR /src
COPY src ./src
RUN javac -d out src/*.java
FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
COPY --from=build /src/out ./out
EXPOSE 8080
ENTRYPOINT ["java", "-cp", "out", "Main"]
