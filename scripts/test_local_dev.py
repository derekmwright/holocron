import importlib.util
from pathlib import Path
import json
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("local_dev", Path(__file__).with_name("local-dev.py"))
local_dev = importlib.util.module_from_spec(spec)
spec.loader.exec_module(local_dev)


class CredentialPersistenceTest(unittest.TestCase):
    def test_atomic_private_credentials_preserve_existing_profiles(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "credentials.json"
            profiles = {"worker": {"token": "test-only-worker"}}
            local_dev.save_private_json(path, profiles)
            profiles["curator"] = {"token": "test-only-curator"}
            local_dev.save_private_json(path, profiles)
            self.assertEqual(json.loads(path.read_text()), profiles)
            self.assertEqual(path.stat().st_mode & 0o777, 0o600)
            self.assertEqual(list(Path(directory).iterdir()), [path])
