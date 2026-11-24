import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("analysis", Path(__file__).with_name("analyze-experiment.py"))
analysis = importlib.util.module_from_spec(spec)
spec.loader.exec_module(analysis)


class AnalysisTest(unittest.TestCase):
    def test_failures_are_charged_and_missing_results_remain_visible(self):
        attempts = [{"id": str(i), "body": {"model": "fixed", "harness": "fixed", "task_family": "spill", "arm": "C"}} for i in range(3)]
        outcomes = [
            {"id": "good", "body": {"attempt": "0", "outcome": "completed", "tokens": 100, "elapsed_ms": 20, "cost_micros": 1}},
            {"id": "bad", "body": {"attempt": "1", "outcome": "failed", "tokens": 300, "elapsed_ms": 40, "cost_micros": 3}},
        ]
        scores = {"good": {"correct": True, "reviewer": "independent"}, "bad": {"correct": False, "reviewer": "independent"}}
        group = analysis.summarize({"attempts": attempts, "outcomes": outcomes}, scores)["groups"][0]
        self.assertEqual(group["tokens_reported"], 400)
        self.assertEqual(group["validated_success_fraction_all_attempts"], 1 / 3)
        self.assertEqual(group["missing_outcomes"], 1)
        self.assertIsNone(group["tokens_per_validated_success"])
        group = analysis.summarize({"attempts": attempts[:2], "outcomes": outcomes}, scores)["groups"][0]
        self.assertEqual(group["tokens_per_validated_success"], 400)

    def test_unscored_completion_is_not_correctness(self):
        data = {"attempts": [{"id": "a", "body": {"model": "m", "harness": "h", "task_family": "f", "arm": "C"}}], "outcomes": [{"id": "o", "body": {"attempt": "a", "outcome": "completed", "tokens": 1, "elapsed_ms": 1, "cost_micros": 0}}]}
        group = analysis.summarize(data, {})["groups"][0]
        self.assertEqual(group["correct"], 0)
        self.assertEqual(group["unscored_outcomes"], 1)


if __name__ == "__main__":
    unittest.main()
