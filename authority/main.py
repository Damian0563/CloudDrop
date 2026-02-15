from confluent_kafka import TopicPartition
from confluent_kafka.admin import AdminClient, NewTopic
from fastapi import FastAPI
from confluent_kafka import Producer, Consumer
import time
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


@app.post("/drop")
async def drop(data: dict):
    try:
        create_expiring_topic(data['key'])
        producer.produce(topic=data['key'], value=data['url'].encode('utf-8'))
        producer.flush()
        return {"status": "ok", "error": ""}
    except Exception as e:
        return {"status": "error", "error": str(e)}


@app.get("/receive/{topic}")
async def receive(topic: str):
    try:
        if is_topic_empty(topic):
            return {"status": "error", "error": "No data found or code expired.", "msg": ""}
        tp = TopicPartition(topic, 0)
        consumer.assign([tp])
        msg = None
        for _ in range(5):
            msg = consumer.poll(timeout=2.0)
            if msg is not None:
                break
        if msg is None:
            return {"status": "error", "error": "No data found or code expired.", "msg": ""}
        if msg.error():
            return {"status": "error", "error": str(msg.error()), "msg": ""}
        admin_client.delete_topics([topic])
        return {"status": "ok", "error": "", "msg": msg.value().decode('utf-8')}
    except Exception as e:
        return {"status": "error", "error": str(e), "msg": ""}
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


def safe_cleanup(timeout=900):
    time.sleep(timeout)
    metadata = admin_client.list_topics(timeout=10)
    all_topics = metadata.topics.keys()
    for topic_name in all_topics:
        if topic_name.startswith('_') or topic_name == 'default':
            continue
        try:
            low, high = consumer.get_watermark_offsets(
                TopicPartition(topic_name, 0))
            if high <= low:
                print(f"Cleanup: Topic {topic_name} is empty. Deleting...")
                admin_client.delete_topics([topic_name])
            else:
                print(f"Cleanup: Topic {
                      topic_name} still has active messages. Skipping.")
        except Exception as e:
            print(f"Could not check topic {topic_name}: {e}")
    safe_cleanup(900)


safe_cleanup()
