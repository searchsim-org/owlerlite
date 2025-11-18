
FROM node:20-alpine
WORKDIR /app
COPY package.json package-lock.json* pnpm-lock.yaml* yarn.lock* ./
RUN npm i --silent || true
COPY . .
RUN npm run build || true
EXPOSE 3000
CMD ["npm","run","start"]
