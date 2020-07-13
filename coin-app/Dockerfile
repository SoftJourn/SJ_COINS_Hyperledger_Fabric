FROM node:12.18.0
WORKDIR /app
COPY package.json /app
RUN npm install
COPY . /app

RUN mkdir -p "/app_data/wallet" && chmod -R 777 "/app_data"

ENTRYPOINT node app.js
EXPOSE 4000
