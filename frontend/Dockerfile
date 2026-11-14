FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_APP_NAME
ARG VITE_DEMO_MODE
ARG VITE_ENABLE_ADMIN
ENV VITE_APP_NAME=$VITE_APP_NAME \
    VITE_DEMO_MODE=$VITE_DEMO_MODE \
    VITE_ENABLE_ADMIN=$VITE_ENABLE_ADMIN
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
