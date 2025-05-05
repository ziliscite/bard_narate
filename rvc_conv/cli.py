import argparse
import shutil
import os
from app import PITCH_ALGO_OPT, process_audio

def main():
    parser = argparse.ArgumentParser(description='RVC Audio Conversion Pipeline')
    parser.add_argument('-i', '--input', required=True, help='Input audio file path')
    parser.add_argument('-o', '--output', required=True, help='Output audio file path')
    parser.add_argument('-m', '--model', required=True, help='Path to RVC model (.pth)')
    parser.add_argument('--index', help='Path to index file')
    parser.add_argument('--pitch_algo', choices=PITCH_ALGO_OPT, default='rmvpe+', help='Pitch extraction algorithm')
    parser.add_argument('--pitch', type=int, default=0, help='Pitch adjustment level (-24 to 24)')
    parser.add_argument('--index_inf', type=float, default=0.75, help='Index influence (0.0 to 1.0)')
    parser.add_argument('--respiration', type=int, default=3, help='Respiration median filter (0-7)')
    parser.add_argument('--envelope', type=float, default=0.25, help='Envelope ratio (0.0 to 1.0)')
    parser.add_argument('--consonant', type=float, default=0.5, help='Consonant protection (0.0 to 0.5)')
    parser.add_argument('--denoise', action='store_true', help='Apply noise reduction')
    parser.add_argument('--effects', action='store_true', help='Apply audio effects')

    args = parser.parse_args()

    processed_files = process_audio(
        audio_files=[args.input],
        model_path=args.model,
        pitch_algo=args.pitch_algo,
        pitch_level=args.pitch,
        index_path=args.index,
        index_influence=args.index_inf,
        respiration_filter=args.respiration,
        envelope_ratio=args.envelope,
        consonant_protection=args.consonant,
        denoise=args.denoise,
        effects=args.effects
    )

    if processed_files:
        os.makedirs(os.path.dirname(args.output), exist_ok=True)
        shutil.copy(processed_files[0], args.output)
        print(f"Successfully created output at {args.output}")

if __name__ == "__main__":
    # py app.py -i "input/input.wav" -o "output.wav" -m "./models/model.pth" --index "./models/model.index" --denoise --effects --pitch '3'
    main()