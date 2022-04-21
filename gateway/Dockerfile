FROM openjdk:11
WORKDIR /app
COPY build/libs/gateway-1.0.war /app
COPY connection.json /app
RUN mkdir -p "/app_data/wallet" && chmod -R 777 "/app_data" && chmod 777 "/app/gateway-1.0.war"

ENTRYPOINT java -jar "/app/gateway-1.0.war"
EXPOSE 4000
