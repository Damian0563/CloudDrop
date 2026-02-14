from fastapi import FastAPI
from confluent_kafka import Producer
import json
import os
from dotenv import load_dotenv
load_dotenv()

app = FastAPI()
producer = Producer({
    'bootstrap.servers': os.getenv('BOOTSTRAP_SERVER'),
    'sasl.mechanisms': 'PLAIN',
    'sasl.username': os.getenv('KAFKA_API_KEY'),
    'sasl.password': os.getenv('KAFKA_API_SECRET'),
    'security.protocol': 'SASL_SSL',
})


@app.post("/drop")
async def drop(data: dict):
    try:
        print(data)
        return {"status": "ok"}
    except Exception as e:
        return {"error": str(e)}


@app.get("/receive/{topic}")
async def receive(topic: str):
    try:
        print(topic)
        return {"status": "ok"}
    except Exception as e:
        return {"error": str(e)}
