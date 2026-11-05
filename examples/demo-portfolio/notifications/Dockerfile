FROM maven:3.9-eclipse-temurin-11 AS build
COPY . /src
RUN mvn -f /src/pom.xml -q package
FROM eclipse-temurin:11-jre
COPY --from=build /src/target/notifications.jar /app.jar
ENV SPRING_PROFILES_ACTIVE=prod
EXPOSE 8080
USER 1001
HEALTHCHECK CMD curl -f http://localhost:8080/actuator/health || exit 1
ENTRYPOINT ["java","-jar","/app.jar"]
