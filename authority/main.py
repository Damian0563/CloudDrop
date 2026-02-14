from confluent_kafka.admin import AdminClient, NewTopic
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

admin_client = AdminClient({
    'bootstrap.servers': os.getenv('BOOTSTRAP_SERVER'),
    'sasl.username': os.getenv('KAFKA_API_KEY'),
    'sasl.password': os.getenv('KAFKA_API_SECRET'),
    'security.protocol': 'SASL_SSL',
    'sasl.mechanisms': 'PLAIN',
})


def create_expiring_topic(topic_name):
    retention_ms = "300000"
    new_topic = NewTopic(
        topic_name,
        num_partitions=1,
        replication_factor=3,
        config={
            'retention.ms': retention_ms,
            'file.delete.delay.ms': '60000'
        }
    )
    fs = admin_client.create_topics([new_topic])
    for topic, f in fs.items():
        try:
            f.result()
            print(f"Topic {topic} created with 15m timeout")
        except Exception as e:
            print(f"Failed to create topic {topic}: {e}")


@app.post("/drop")
async def drop(data: dict):
    try:
        create_expiring_topic(data['key'])
        producer.produce(topic=data['key'], value=json.dumps(
            data['url']).encode('utf-8'))
        remaining = producer.flush(3)
        if remaining > 0:
            return {"error": "Failed to send message to broker."}
        return {"ok": "ok"}
    except Exception as e:
        return {"error": str(e)}


@app.get("/receive/{topic}")
async def receive(topic: str):
    try:
        print(topic)
        return {"ok": "ok"}
    except Exception as e:
        return {"error": str(e)}
