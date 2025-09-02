# How to run your Java app with Guvnor

## Setup

```bash
cd my-java-app
mvn clean package
guvnor init
```

## Configuration

```yaml
# guvnor.yaml
server:
  http_port: 8080
  https_port: 8443
  log_level: info

apps:
  - name: java-app
    hostname: localhost
    port: 3000
    command: java
    args:
      - "-jar"
      - "target/myapp.jar"
      - "--server.port=3000"
    working_dir: /path/to/your/java-app
    environment:
      JAVA_OPTS: "-Xms512m -Xmx1024m"
      SPRING_PROFILES_ACTIVE: "production"
    health_check:
      enabled: true
      path: /actuator/health
      interval: 30s
    restart_policy:
      enabled: true
      max_retries: 5
      backoff: 5s

tls:
  enabled: false
  auto_cert: false
  cert_dir: ./certs
```

## Usage

```bash
# Production deployment
mvn clean package               # Build first
guvnor start                    # Access: http://localhost:8080/
guvnor logs                     # View logs
guvnor status                   # Check status
guvnor stop                     # Stop app

# Development
guvnor start                    # If using dev config below
```

## Alternative Configurations

### Spring Boot Application
```yaml
apps:
  - name: spring-boot
    command: java
    args: ["-jar", "target/myapp.jar"]
    environment:
      JAVA_OPTS: "-Xms256m -Xmx512m"
      SPRING_PROFILES_ACTIVE: "prod"
```

### Development Mode
```yaml
apps:
  - name: java-dev
    command: mvn
    args: ["spring-boot:run"]
    environment:
      SPRING_PROFILES_ACTIVE: "development"
```

### With JVM Arguments
```yaml
apps:
  - name: java-tuned
    command: java
    args:
      - "-XX:+UseG1GC"
      - "-XX:MaxGCPauseMillis=200"
      - "-Xms1g"
      - "-Xmx2g"
      - "-jar"
      - "target/myapp.jar"
```

### Gradle Application
```yaml
apps:
  - name: gradle-app
    command: java
    args: ["-jar", "build/libs/myapp.jar"]
    environment:
      JAVA_OPTS: "-server"
```

## Build Commands

```bash
# Maven build
mvn clean package
mvn clean package -DskipTests

# Gradle build
gradle clean build
gradle clean bootJar

# Create executable JAR
mvn clean package spring-boot:repackage
```

## Required pom.xml (Maven)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    
    <groupId>com.example</groupId>
    <artifactId>myapp</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>
    
    <properties>
        <java.version>17</java.version>
        <spring.boot.version>3.1.0</spring.boot.version>
    </properties>
    
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
    </dependencies>
    
    <build>
        <plugins>
            <plugin>
                <groupId>org.springframework.boot</groupId>
                <artifactId>spring-boot-maven-plugin</artifactId>
            </plugin>
        </plugins>
    </build>
</project>
```

## Java Requirements

- Java 17 or higher
- Maven 3.6+ or Gradle 7+
- Spring Boot 3.x for web applications