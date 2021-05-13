import argparse
import sys

"""
Overrides argparse.ArgumentParser so that it emits error messages to
stdout instead of stderr.
"""
class MyArgumentParser(argparse.ArgumentParser):
    def _print_message(self, message, fd=None):
        if message:
            fd = sys.stdout
            fd.write(message)
