FROM node:20-alpine

WORKDIR /app

# Install global qodercli dependencies required by the proxy
RUN npm install -g @qodercn-ai/qoderclicn @qoder-ai/qodercli

COPY package*.json ./
RUN npm install --production

COPY . .

EXPOSE 3000

# Set environment variables with defaults if needed
ENV HOST=0.0.0.0
ENV PORT=3000

CMD ["npm", "start"]
