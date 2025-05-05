import os
import json
import logging
import tempfile
import pika
import boto3
from pathlib import Path
from pika import PlainCredentials
from typing import Dict, Any
from kokoro import KPipeline
from inference import Inference

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class Config:
    """Central configuration management"""
    
    def __init__(self):
        self.rabbitmq_host = os.getenv("RABBITMQ_HOST", "localhost")
        self.rabbitmq_port = int(os.getenv("RABBITMQ_PORT", "5672"))
        self.rabbitmq_user = os.getenv("RABBITMQ_USER", "guest")
        self.rabbitmq_pass = os.getenv("RABBITMQ_PASSWORD", "guest")
        self.exchange_name = os.getenv("RABBITMQ_EXCHANGE", "processing_exchange")
        self.input_queue = os.getenv("RABBITMQ_INPUT_QUEUE", "s3_processing_queue")
        self.output_queue = os.getenv("RABBITMQ_OUTPUT_QUEUE", "s3_converting_queue")
        self.input_routing_key = os.getenv("RABBITMQ_INPUT_ROUTING_KEY", "s3_file_key")
        self.output_routing_key = os.getenv("RABBITMQ_OUTPUT_ROUTING_KEY", "processed_file_key")

        self.aws_access_key = os.getenv("AWS_ACCESS_KEY_ID")
        self.aws_secret_key = os.getenv("AWS_SECRET_ACCESS_KEY")
        self.aws_region = os.getenv("AWS_REGION", "us-east-1")
        self.s3_bucket = os.getenv("S3_BUCKET_NAME")
        self.processed_prefix = os.getenv("S3_PROCESSED_PREFIX", "processed/")

        self._validate()

    def _validate(self):
        if not all([self.aws_access_key, self.aws_secret_key, self.s3_bucket]):
            raise ValueError("Missing required AWS configuration")
        if not self.input_queue:
            raise ValueError("RabbitMQ input queue not configured")

class S3Client:
    """Encapsulates S3 operations with temporary file handling"""
    
    def __init__(self, config: Config):
        self.config = config
        self._client = boto3.client(
            "s3",
            aws_access_key_id=config.aws_access_key,
            aws_secret_access_key=config.aws_secret_key,
            region_name=config.aws_region
        )

    def download_to_tempfile(self, key: str) -> tempfile._TemporaryFileWrapper[bytes]:
        """Download S3 object to a temporary file"""
        try:
            temp_file = tempfile.NamedTemporaryFile(delete = False)

            self._client.download_fileobj(
                Bucket=self.config.s3_bucket,
                Key=key,
                Fileobj=temp_file
            )

            temp_file.flush()
            logger.info(f"Downloaded {key} to {temp_file.name}")

            return temp_file
        
        except Exception as e:
            temp_file.close()

            # delete temp file
            os.unlink(temp_file.name)
            raise RuntimeError(f"Failed to download {key}: {str(e)}") from e

    def upload_from_tempfile(self, temp_file: tempfile._TemporaryFileWrapper[bytes], key: str) -> str:
        """Upload the processed file to S3 and return new key"""
        try:
            # Either differentiate through prefix or through different bucket
            # doing both for now
            processed_key = f"{self.config.processed_prefix}{os.path.basename(key)}"
            temp_file.seek(0)

            self._client.upload_fileobj(
                Fileobj=temp_file,
                Bucket=self.config.s3_bucket,
                Key=processed_key
            )

            logger.info(f"Uploaded processed file to {processed_key}")
            return processed_key
        
        finally:
            temp_file.close()
            os.unlink(temp_file.name)

class RabbitMQClient:
    """Handles RabbitMQ connection and messaging"""
    
    def __init__(self, config: Config):
        self.config = config
        self._connection = None
        self._channel = None
        self._connect()

    def _connect(self):
        """Establish RabbitMQ connection and channel"""
        credentials = PlainCredentials(
            username=self.config.rabbitmq_user,
            password=self.config.rabbitmq_pass
        )

        parameters = pika.ConnectionParameters(
            host=self.config.rabbitmq_host,
            port=self.config.rabbitmq_port,
            credentials=credentials
        )

        self._connection = pika.BlockingConnection(parameters)
        self._channel = self._connection.channel()
        self._setup()

    def _setup(self):
        """Declare exchange and queues"""
        self._channel.exchange_declare(
            exchange=self.config.exchange_name,
            exchange_type="topic",
            durable=True
        )

        # input, text
        self._channel.queue_declare(
            queue=self.config.input_queue,
            durable=True
        )

        # output, mp3
        self._channel.queue_declare(
            queue=self.config.output_queue,
            durable=True
        )

        # bind to file.text
        self._channel.queue_bind(
            exchange=self.config.exchange_name,
            queue=self.config.input_queue,
            routing_key=self.config.input_routing_key
        )

    def consume_messages(self, callback):
        """Start consuming messages with the given callback"""
        self._channel.basic_qos(prefetch_count=1)
        self._channel.basic_consume(
            queue=self.config.input_queue,
            on_message_callback=callback,
            auto_ack=False
        )

        logger.info("Started consuming messages...")
        self._channel.start_consuming()

    def publish_message(self, message: Dict[str, Any]):
        """Publish a message to the configured exchange"""
        self._channel.basic_publish(
            exchange=self.config.exchange_name,
            routing_key=self.config.output_routing_key,
            body=json.dumps(message),
            properties=pika.BasicProperties(
                delivery_mode=2,
                content_type="application/json"
            )
        )

    def close(self):
        """Close the connection gracefully"""
        if self._connection and self._connection.is_open:
            self._channel.close()
            self._connection.close()
            logger.info("RabbitMQ connection closed")

# bikin job consumer, dia consume di semua topic, routing key file.#. ambil id sama status
# tiap consume, ya update ae job dg id yang masuk sesuai dengan status.
#
# publisher macam di gateway tuh, yg publish text file, nah itu kan ke file.text
# tinggal taro aja tambahan "status": "proccessing" pas dia publish ke kokoro
# ntar diconsume kokoro sama job service
#
# edan, cerdas coy

class FileProcessor:
    """Orchestrates file processing workflow"""
    
    def __init__(self, s3_client: S3Client, mq_client: RabbitMQClient, infer: Inference):
        self.s3_client = s3_client
        self.mq_client = mq_client
        self.inference = infer

    def _process_message(self, ch, method, properties, body):
        """Handle incoming message processing"""
        try:
            message = json.loads(body)
            original_key = message["file_key"]

            job_id = message["job_id"] # do something w ts

            logger.info(f"Processing file: {original_key}")

            # Temp file is the input file
            with self.s3_client.download_to_tempfile(original_key) as temp_file:
                # Process the file
                outfile = self._process_file(temp_file)

                # Upload processed file and get new key
                processed_key = self.s3_client.upload_from_tempfile(
                    outfile, original_key
                )

                # Publish result
                # Since it has been processed, we can update the job status to Converting
                self.mq_client.publish_message({"job_id": job_id, "job_status": "Converting", "file_key": processed_key})

            ch.basic_ack(delivery_tag=method.delivery_tag)
            logger.info(f"Completed processing {original_key}")

        except Exception as e:
            logger.error(f"Error processing {message.get('key')}: {str(e)}", exc_info=True)
            ch.basic_ack(delivery_tag=method.delivery_tag)

    def _process_file(self, temp_file) -> tempfile._TemporaryFileWrapper[bytes]:
        """Convert text into audio and store it. Return absolute filepath"""
        temp_out_file = tempfile.NamedTemporaryFile(suffix=".wav", delete=False)

        self.inference.generate(temp_out_file.name, open(temp_file.name, 'r').read())
        return temp_out_file

    def start(self):
        """Start the processing loop"""
        self.mq_client.consume_messages(self._process_message)

def main() -> None:
    try:
        config = Config()
        s3_client = S3Client(config)
        mq_client = RabbitMQClient(config)

        pipeline = KPipeline(lang_code="a", trf=True)
        infer = Inference(pipeline, "output")

        processor = FileProcessor(s3_client, mq_client, infer)
        processor.start()

    except Exception as e:
        logger.error(f"error: {e}", exc_info=True)
        # mq_client.publish_message pub to job service if failed for some reason
        mq_client.close()

if __name__ == "__main__":
    main()
