#!/usr/bin/env python3
import argparse
import os
from typing import Any, Generator
from pathlib import Path
import numpy as np
import torch
import soundfile as sf
from kokoro import KPipeline

class Inference: 
    def __init__(self, pipeline: KPipeline, output_dir: str):
        self._pipeline = pipeline
        self.output = output_dir
        os.makedirs(self.output, exist_ok=True)
    
    def __generate(
        self, text: str, voice: str, speed: float
    ) -> Generator["KPipeline.Result", None, None]:
        yield from self._pipeline(text, voice=voice, speed=speed, split_pattern=r"\n+")

    def generate(
        self, output_file: Path, text: str, voice: str | None, speed: float = 1.0
    ) -> None:
        """
        Generate complete audio file from text string.
        """
        if voice is None:
            voice = "af_bella"
    
        # output_file must have extention, like .wav
        with sf.SoundFile(str(output_file.resolve()), mode='w', samplerate=24000, channels=1, subtype='PCM_16') as sf_file:
            for result in self.__generate(text, voice=voice, speed=speed):
                if result.audio is None:
                    continue

                try:
                    # Convert the generated audio to int16 PCM format
                    audio_data = (result.audio.numpy() * 32767).astype(np.int16)
                except Exception as e:
                    continue

                try:
                    sf_file.write(audio_data)
                except Exception as e:
                    raise e
        
    def infer(
        self,  
        text: Any | str, 
        voice: str | None, 
        speed: float, 
        name: str
    ) -> str:
        """
        Generate complete audio file with concatenation and error handling.
        Return output file path.
        """

        if voice is None:
            voice = "af_bella"

        generator = self._pipeline(
            text, 
            voice=voice,
            speed=speed,
            split_pattern=r'\n+'
        )

        audio_segments = []
        try:
            for i, (graphemes, phonemes, audio) in enumerate(generator):
                print(f"Generated segment {i}:")
                print(f"Text: {graphemes}")
                print(f"Phonemes: {phonemes}\n")

                audio_segments.append(audio)
        except Exception as e:
            raise RuntimeError(f"Failed during audio generation: {str(e)}") from e

        if not audio_segments:
            raise RuntimeError("No audio generated - empty input or generation failure")

        try:
            full_audio = np.concatenate(audio_segments)
        except ValueError as e:
            raise RuntimeError("Failed to concatenate audio segments") from e
        
        output_path = os.path.join(self.output, f"{name}.wav")

        try:
            sf.write(output_path, full_audio, 24000)
            return output_path
        except Exception as e:
            raise IOError(f"Failed to write output file: {str(e)}") from e

    def infer_stream(
        self,
        text: Any | str, 
        voice: str | None, 
        speed: float,
        name: str,
        retries: int = 3
    ) -> None:
        generator = self._pipeline(
            text,
            voice=voice,
            speed=speed,
            split_pattern=r'\n+'
        )

        for i, (graphemes, phonemes, audio) in enumerate(generator):
            attempt = 0
            
            print(f"Generated segment {i}:")
            print(f"Text: {graphemes}")
            print(f"Phonemes: {phonemes}\n")

            while attempt <= retries:
                try:
                    sf.write(os.path.join(self.output, f'{name}_{i}.wav'), audio, 24000)
                    print(f"Segment {i} succeeded")
                    break

                except Exception as e:
                    if attempt >= retries:
                        raise Exception(f"Segment {i} failed after {retries} retries: {str(e)}") from e
                    
                    attempt += 1
                    print(f"Retrying segment {i} (attempt {attempt})")

        print(f"\nSuccessfully converted {i+1} segments")

def main():
    parser = argparse.ArgumentParser(description='Kokoro TTS CLI')
    parser.add_argument('--voice', help='Voice name or path to .pt voice tensor file. Default would be af_bella')
    parser.add_argument('--text', help='Input text to synthesize')
    parser.add_argument('--input-file', help='Path to text file to synthesize')
    parser.add_argument('--output-dir', default='output', help='Output WAV directory path')
    parser.add_argument('--speed', type=float, default=1.0, help='Speech speed adjustment')
    parser.add_argument('--stream', default=False, help='Whether output should be concat or remain split per token')

    args = parser.parse_args()

    if not (args.text or args.input_file):
        parser.error('Must provide either --text or --input-file')
    if args.text and args.input_file:
        parser.error('Cannot use both --text and --input-file')

    text = args.text or open(args.input_file, 'r').read()

    voice = ''
    if not args.voice:
        print(f"Using default voice: af_bella")
        voice = "af_bella"
    elif os.path.isfile(args.voice):
        print(f"Loading voice tensor from {args.voice}")
        voice = torch.load(args.voice, weights_only=True)
    else:
        print(f"Using predefined voice: {args.voice}")
        voice = args.voice

    pipeline = KPipeline(lang_code="a", trf=True)
    inference = Inference(pipeline, "output")

    try:
        if args.stream:
            inference.infer_stream(text, voice, args.speed, args.output_dir)
        else:
            inference.generate("yayaya", text, voice, args.speed)
    except Exception as e:
        parser.error(e)
    
if __name__ == '__main__':
    # python app.py --text 'The sky above the port was the color of television, tuned to a dead channel. Case heard someone say, as he shouldered his way through the crowd around the door of the Chat. It was a Sprawl voice and a Sprawl joke. The Chatsubo was a bar for professional expatriates; you could drink there for a week and never hear two words in Japanese.'
    # python app.py --input-file 'C:\Users\manzi\VSCoding\kokoro_cli\input\text.txt' --output-dir 'stream' --stream
    main()