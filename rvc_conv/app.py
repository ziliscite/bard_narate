import random
import logging
from infer_rvc_python import BaseLoader
from pedalboard import Pedalboard, Reverb, Compressor, HighpassFilter
from pedalboard.io import AudioFile
from pydub import AudioSegment
import noisereduce as nr
import numpy as np

logging.getLogger("infer_rvc_python").setLevel(logging.ERROR)

class PostProcessor:
    def add_audio_effects(self, audio_list: list[str]):
        result = []
        for audio_path in audio_list:
            board = Pedalboard([
                HighpassFilter(),
                Compressor(ratio=4, threshold_db=-15),
                Reverb(room_size=0.10, dry_level=0.8, wet_level=0.2, damping=0.7)
            ])

            with AudioFile(audio_path, 'w', f.samplerate, f.num_channels) as f:
                while f.tell() < f.frames:
                    chunk = f.read(int(f.samplerate))
                    effected = board(chunk, f.samplerate, reset=False)
                    f.write(effected)

            result.append(audio_path)

        return result

    def apply_noisereduce(self, audio_list: list[str]):
        result = []
        for audio_path in audio_list:
            audio = AudioSegment.from_file(audio_path)
            samples = np.array(audio.get_array_of_samples())
            reduced_noise = nr.reduce_noise(samples, sr=audio.frame_rate, prop_decrease=0.6)
            reduced_audio = AudioSegment(
                reduced_noise.tobytes(),
                frame_rate=audio.frame_rate,
                sample_width=audio.sample_width,
                channels=audio.channels
            )

            reduced_audio.export(audio_path, format="wav")
            result.append(audio_path)

        return result

class VoiceConverter:
    """Handles voice conversion using RVC models"""
    PITCH_ALGO_OPT = [
        "pm",
        "harvest",
        "crepe",
        "rmvpe",
        "rmvpe+",
    ]

    def __init__(self, loader: BaseLoader, processor: PostProcessor):
        self._converter = loader
        self._processor = processor

    def _convert_now(self, audio_files, random_tag):
        return self._converter(audio_files, random_tag, overwrite=True, parallel_workers=8)

    def process_audio(
            self,
            audio_files: str | list[str],
            model_path: str,
            pitch_algo: str ="rmvpe+",
            pitch_level: int = 0,
            index_path: str | None = None,
            index_influence: float = 0.75,
            respiration_filter: int = 3,
            envelope_ratio: float = 0.25,
            consonant_protection: float = 0.5,
            denoise: bool = False,
            effects: bool = False
    ):
        if not audio_files:
            raise ValueError("No audio files provided")

        if isinstance(audio_files, str):
            audio_files = [audio_files]

        random_tag = f"USER_{random.randint(10000000, 99999999)}"

        self._converter.apply_conf(
            tag=random_tag,
            file_model=model_path,
            pitch_algo=pitch_algo,
            pitch_lvl=pitch_level,
            file_index=index_path,
            index_influence=index_influence,
            respiration_median_filtering=respiration_filter,
            envelope_ratio=envelope_ratio,
            consonant_breath_protection=consonant_protection,
            resample_sr=44100 if audio_files[0].endswith('.mp3') else 0,
        )

        result = self._convert_now(audio_files, random_tag)

        # try:
        #     if denoise:
        #         result = self._processor.apply_noisereduce(result)
        #     if effects:
        #         result = self._processor.add_audio_effects(result)
        # except Exception as e:
        #     print(e)
        # TODO: Later try except in the consumer
        # if error, then we must notify the job service

        if denoise:
            result = self._processor.apply_noisereduce(result)
        if effects:
            result = self._processor.add_audio_effects(result)

        return result