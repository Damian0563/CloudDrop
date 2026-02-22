from confluent_kafka import TopicPartition
from datetime import timezone
from confluent_kafka.admin import AdminClient, NewTopic
from fastapi import FastAPI, status,  HTTPException, Depends
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from typing import Annotated
from confluent_kafka import Producer, Consumer
import threading
import datetime
import time
import os
import json
from dotenv import load_dotenv
from google.cloud import storage
import uvicorn
load_dotenv()
os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = os.environ[
    'CREDENTIALS_PATH']
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
consumer = Consumer({
    'bootstrap.servers': os.getenv('BOOTSTRAP_SERVER'),
    'sasl.username': os.getenv('KAFKA_API_KEY'),
    'sasl.password': os.getenv('KAFKA_API_SECRET'),
    'security.protocol': 'SASL_SSL',
    'sasl.mechanisms': 'PLAIN',
    'group.id': 'test-group',
    'auto.offset.reset': 'earliest',
    'enable.auto.commit': True,
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


security = HTTPBearer()


async def validate_token(credentials: Annotated[HTTPAuthorizationCredentials, Depends(security)]):
    token = credentials.credentials
    if token != os.getenv('SECRET'):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Unauthorized request."
        )
    return True


@app.post("/drop")
async def drop(data: dict, authorized: bool = Depends(validate_token)):
    try:
        create_expiring_topic(data['key'])
        payload = {
            "url": data['url'],
            "original_name": data.get('original_name', ''),
            "is_dir": data.get('is_dir', False)
        }
        producer.produce(topic=data['key'],
                         value=json.dumps(payload).encode('utf-8'))
        producer.flush()
        return {"status": "ok", "error": ""}
    except Exception as e:
        return {"status": "error", "error": str(e)}


@app.get("/receive/{topic}")
async def receive(topic: str, authorized: bool = Depends(validate_token)):
    try:
        if is_topic_empty(topic):
            return {"status": "error", "error": "No data found or code expired.", "msg": "", "original_name": "", "is_dir": False}
        tp = TopicPartition(topic, 0)
        consumer.assign([tp])
        msg = None
        for _ in range(5):
            msg = consumer.poll(timeout=2.0)
            if msg is not None:
                break
        if msg is None:
            return {"status": "error", "error": "No data found or code expired.", "msg": "", "original_name": "", "is_dir": False}
        if msg.error():
            return {"status": "error", "error": str(msg.error()), "msg": "", "original_name": "", "is_dir": False}
        admin_client.delete_topics([topic])
        data = json.loads(msg.value().decode('utf-8'))
        return {
            "status": "ok",
            "error": "",
            "msg": data.get('url', ''),
            "original_name": data.get('original_name', ''),
            "is_dir": data.get('is_dir', False)
        }
    except Exception as e:
        return {"status": "error", "error": str(e), "msg": "", "original_name": "", "is_dir": False}
    finally:
        consumer.unassign()


def is_topic_empty(topic_name):
    try:
        low, high = consumer.get_watermark_offsets(
            TopicPartition(topic_name, 0))
        if high <= low:
            return True
        return False
    except Exception:
        return True


def cleanup_gcs():
    storage_client = storage.Client()
    bucket = storage_client.bucket(os.environ['BUCKET_NAME'])
    blobs = bucket.list_blobs()
    now = datetime.datetime.now(timezone.utc)
    for blob in blobs:
        age = now - blob.time_created
        if age > datetime.timedelta(minutes=15):
            blob.delete()


def cleanup_kafka():
    metadata = admin_client.list_topics(timeout=10)
    all_topics = metadata.topics.keys()
    for topic_name in all_topics:
        if topic_name.startswith('_') or topic_name == 'default':
            continue
        try:
            low, high = consumer.get_watermark_offsets(
                TopicPartition(topic_name, 0), timeout=2.0)
            if high <= low:
                try:
                    admin_client.delete_topics([topic_name])
                except Exception:
                    continue
        except Exception as e:
            print(f"Could not check topic {topic_name}: {e}")


def run_continuously():
    while True:
        try:
            print("Running cleanup cycle...")
            cleanup_gcs()
            cleanup_kafka()
        except Exception as e:
            print(f"Cleanup error: {e}")
        time.sleep(600)


if __name__ == "__main__":
    cleanup_thread = threading.Thread(target=run_continuously, daemon=True)
    cleanup_thread.start()
    uvicorn.run(app, host="0.0.0.0", port=8000)
