FROM golang:1.26.5 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o efr_bot


FROM alpine:3.24

# Certificates setup
WORKDIR /usr/local/share/ca-certificates/
RUN apk add --no-cache ca-certificates
RUN wget https://gu-st.ru/content/downloads/Russian_Trusted_Root_CA.cer && \
    wget https://gu-st.ru/content/downloads/Russian_Trusted_Sub_CA.cer && \
    wget https://gu-st.ru/content/downloads/Russian_Trusted_Sub_CA_2024.cer
RUN update-ca-certificates

WORKDIR /app

COPY --from=builder /app/efr_bot ./

CMD [ "./efr_bot" ]
