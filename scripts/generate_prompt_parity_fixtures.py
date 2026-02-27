#!/usr/bin/env python3
"""Generate fixed Prompt parity fixtures from Python Agently baseline.

This script is intentionally offline and deterministic:
- no model requests
- no network dependency
- fixed prompt settings (prompt.add_current_time defaults to False)
"""

from __future__ import annotations

import copy
import json
import os
import shutil
import sys
from pathlib import Path
from typing import Any


PYTHON_BASELINE = Path(os.environ.get("AGENTLY_PYTHON_BASELINE", "/path/to/Agently")).expanduser()
REPO_ROOT = Path(__file__).resolve().parent.parent
FIXTURE_DIR = REPO_ROOT / "tests" / "fixtures" / "prompt_parity"


def _bootstrap_python_baseline() -> None:
    if not PYTHON_BASELINE.exists():
        raise RuntimeError(f"python baseline not found: {PYTHON_BASELINE}")
    sys.path.insert(0, str(PYTHON_BASELINE))


def _normalize_line_ending(text: str) -> str:
    return text.replace("\r\n", "\n").replace("\r", "\n")


def _apply_prompt_data(agent: Any, prompt_data: dict[str, Any]) -> None:
    for key, value in prompt_data.get(".agent", {}).items():
        agent.set_agent_prompt(key, value)
    for key, value in prompt_data.get(".request", {}).items():
        agent.set_request_prompt(key, value)


def _apply_settings(agent: Any, settings_map: dict[str, Any]) -> None:
    for key in sorted(settings_map.keys()):
        agent.set_settings(key, settings_map[key])


def _collect_expected(
    agent: Any,
    message_options: dict[str, Any],
) -> tuple[dict[str, Any], dict[str, Any]]:
    request_prompt = agent.request_prompt

    expected: dict[str, Any] = {}
    errors: dict[str, Any] = {}

    try:
        text = request_prompt.to_text()
        expected["expected_text"] = _normalize_line_ending(text)
    except Exception as exc:  # pragma: no cover - captured into fixture
        errors["expected_text_error"] = str(exc)

    try:
        expected["expected_messages"] = request_prompt.to_messages(
            rich_content=bool(message_options.get("rich_content", False)),
            strict_role_orders=bool(message_options.get("strict_role_orders", True)),
        )
    except Exception as exc:  # pragma: no cover - captured into fixture
        errors["expected_messages_error"] = str(exc)

    serializable_request = request_prompt.to_serializable_prompt_data(inherit=False)
    expected["expected_output_schema"] = serializable_request.get("output")
    expected["expected_serializable_prompt"] = {
        ".agent": agent.agent_prompt.to_serializable_prompt_data(inherit=False),
        ".request": serializable_request,
    }
    return expected, errors


def _build_cases() -> list[dict[str, Any]]:
    # keep object keys in alphabetical order whenever possible to avoid map-order ambiguity.
    return [
        {
            "case_id": "prompt_001_empty_prompt_error",
            "mode": "direct",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "prompt_data": {".agent": {}, ".request": {}},
        },
        {
            "case_id": "prompt_002_input_only",
            "mode": "direct",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "prompt_data": {".agent": {}, ".request": {"input": "hello"}},
        },
        {
            "case_id": "prompt_003_full_slots_with_json_output",
            "mode": "direct",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "prompt_data": {
                ".agent": {
                    "developer": "developer directions",
                    "system": "system role",
                },
                ".request": {
                    "examples": ["ex1", "ex2"],
                    "info": {"a": "A", "b": "B"},
                    "input": "ask",
                    "instruct": ["do-1", "do-2"],
                    "output": {
                        "answer": {"$desc": "final answer", "$type": "str"},
                        "steps": [{"$desc": "one step", "$type": "str"}],
                    },
                },
            },
        },
        {
            "case_id": "prompt_004_output_format_markdown_manual",
            "mode": "direct",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "prompt_data": {
                ".agent": {},
                ".request": {
                    "input": "say hi",
                    "output": {"answer": {"$type": "str"}},
                    "output_format": "markdown",
                },
            },
        },
        {
            "case_id": "prompt_005_output_format_text_manual",
            "mode": "direct",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "prompt_data": {
                ".agent": {},
                ".request": {
                    "input": "say hi",
                    "output": {"answer": {"$type": "str"}},
                    "output_format": "text",
                },
            },
        },
        {
            "case_id": "prompt_006_role_mapping_override",
            "mode": "direct",
            "settings": {
                "prompt.role_mapping": {
                    "_": "assistant",
                    "assistant": "assistant",
                    "developer": "developer",
                    "system": "system",
                    "user": "user",
                },
            },
            "message_options": {
                "rich_content": False,
                "role_mapping": {"assistant": "assistant", "user": "user"},
                "strict_role_orders": True,
            },
            "prompt_data": {
                ".agent": {
                    "system": "You are mapped",
                },
                ".request": {"input": "hello"},
            },
        },
        {
            "case_id": "prompt_007_prompt_title_mapping_override",
            "mode": "direct",
            "settings": {
                "prompt.prompt_title_mapping": {
                    "action_results": "ACTION RESULTS",
                    "chat_history": "CHAT HISTORY",
                    "developer": "DEVELOPER",
                    "examples": "EXAMPLES",
                    "info": "INFO",
                    "input": "INPUT",
                    "instruct": "INSTRUCTION",
                    "output": "OUTPUT",
                    "output_requirement": "OUTPUT REQUIREMENT",
                    "system": "SYSTEM ROLE",
                    "tools": "TOOLS",
                }
            },
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "prompt_data": {
                ".agent": {"system": "sys"},
                ".request": {"input": "hello", "output": {"answer": {"$type": "str"}}},
            },
        },
        {
            "case_id": "prompt_008_chat_history_strict_false",
            "mode": "direct",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": False},
            "prompt_data": {
                ".agent": {},
                ".request": {
                    "chat_history": [
                        {"content": "A1", "role": "assistant"},
                        {"content": "A2", "role": "assistant"},
                        {"content": "U1", "role": "user"},
                    ],
                    "input": "Q",
                },
            },
        },
        {
            "case_id": "prompt_009_chat_history_strict_true",
            "mode": "direct",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "prompt_data": {
                ".agent": {},
                ".request": {
                    "chat_history": [
                        {"content": "A1", "role": "assistant"},
                        {"content": "A2", "role": "assistant"},
                        {"content": "U1", "role": "user"},
                    ],
                    "input": "Q",
                },
            },
        },
        {
            "case_id": "prompt_010_chat_history_rich_content",
            "mode": "direct",
            "settings": {},
            "message_options": {"rich_content": True, "strict_role_orders": True},
            "prompt_data": {
                ".agent": {},
                ".request": {
                    "chat_history": [
                        {
                            "content": [
                                {"text": "hello", "type": "text"},
                                {"image_url": {"url": "http://img"}, "type": "image_url"},
                            ],
                            "role": "assistant",
                        },
                        {"content": [{"text": "question", "type": "text"}], "role": "user"},
                    ],
                    "input": "continue",
                },
            },
        },
        {
            "case_id": "prompt_011_attachment_only_rich_false",
            "mode": "direct",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "prompt_data": {
                ".agent": {},
                ".request": {
                    "attachment": [
                        {"text": "text-a", "type": "text"},
                        {"image_url": {"url": "http://img"}, "type": "image_url"},
                    ]
                },
            },
        },
        {
            "case_id": "prompt_012_attachment_only_rich_true",
            "mode": "direct",
            "settings": {},
            "message_options": {"rich_content": True, "strict_role_orders": True},
            "prompt_data": {
                ".agent": {},
                ".request": {
                    "attachment": [
                        {"text": "text-a", "type": "text"},
                        {"image_url": {"url": "http://img"}, "type": "image_url"},
                    ]
                },
            },
        },
        {
            "case_id": "configure_013_yaml_basic_mappings",
            "mode": "configure",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "configure": {
                "format": "yaml",
                "content": """
.agent:
  system: You are ${role}
.request:
  input: Ask ${topic}
$persona: ${persona}
${request_key}: ${request_value}
""".strip(),
                "mappings": {
                    "persona": "teacher",
                    "request_key": "note",
                    "request_value": "from-yaml",
                    "role": "assistant",
                    "topic": "recursion",
                },
                "prompt_key_path": "",
            },
        },
        {
            "case_id": "configure_014_json_prompt_key_path",
            "mode": "configure",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "configure": {
                "format": "json",
                "content": json.dumps(
                    {
                        "p1": {".request": {"input": "wrong"}},
                        "p2": {
                            ".agent": {"system": "SYS ${name}"},
                            ".request": {"input": "IN ${topic}", "output": {"reply": {"$type": "str"}}},
                        },
                    },
                    ensure_ascii=False,
                ),
                "mappings": {"name": "N1", "topic": "T1"},
                "prompt_key_path": "p2",
            },
        },
        {
            "case_id": "configure_015_output_type_desc_conversion",
            "mode": "configure",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "configure": {
                "format": "yaml",
                "content": """
.request:
  input: test
  output:
    answer:
      $type: str
      $desc: final answer
    extra:
      .type:
        detail:
          $type: str
      .desc: extra block
""".strip(),
                "mappings": {},
                "prompt_key_path": "",
            },
        },
        {
            "case_id": "configure_016_alias_set_request_prompt",
            "mode": "configure",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "configure": {
                "format": "yaml",
                "content": """
.alias:
  set_request_prompt:
    .args:
      - instruct
      - Reply politely.
.request:
  input: hi
""".strip(),
                "mappings": {},
                "prompt_key_path": "",
            },
        },
        {
            "case_id": "configure_017_roundtrip_yaml",
            "mode": "configure_roundtrip_yaml",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "prompt_data": {
                ".agent": {"system": "SYS"},
                ".request": {"input": "IN", "output": {"answer": {"$type": "str"}}},
            },
        },
        {
            "case_id": "configure_018_roundtrip_json",
            "mode": "configure_roundtrip_json",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "prompt_data": {
                ".agent": {"system": "SYS"},
                ".request": {"input": "IN", "output": {"answer": {"$type": "str"}}},
            },
        },
        {
            "case_id": "configure_019_extra_field_order_preserved",
            "mode": "configure",
            "settings": {},
            "message_options": {"rich_content": False, "strict_role_orders": True},
            "configure": {
                "format": "yaml",
                "content": """
$persona: mentor
$style: concise
goal: parity
hint: keep-order
.request:
  input: check order
""".strip(),
                "mappings": {},
                "prompt_key_path": "",
            },
        },
    ]


def _run_case(case: dict[str, Any]) -> dict[str, Any]:
    from agently import Agently

    agent = Agently.create_agent()
    _apply_settings(agent, case.get("settings", {}))

    mode = case["mode"]
    if mode == "direct":
        _apply_prompt_data(agent, case["prompt_data"])
    elif mode == "configure":
        configure = copy.deepcopy(case["configure"])
        if configure["format"] == "yaml":
            agent.load_yaml_prompt(
                configure["content"],
                mappings=configure.get("mappings"),
                prompt_key_path=configure.get("prompt_key_path") or None,
            )
        else:
            agent.load_json_prompt(
                configure["content"],
                mappings=configure.get("mappings"),
                prompt_key_path=configure.get("prompt_key_path") or None,
            )
    elif mode == "configure_roundtrip_yaml":
        _apply_prompt_data(agent, case["prompt_data"])
        yaml_prompt = agent.get_yaml_prompt()
        agent = Agently.create_agent()
        _apply_settings(agent, case.get("settings", {}))
        agent.load_yaml_prompt(yaml_prompt)
    elif mode == "configure_roundtrip_json":
        _apply_prompt_data(agent, case["prompt_data"])
        json_prompt = agent.get_json_prompt()
        agent = Agently.create_agent()
        _apply_settings(agent, case.get("settings", {}))
        agent.load_json_prompt(json_prompt)
    else:
        raise RuntimeError(f"unknown mode: {mode}")

    expected, errors = _collect_expected(agent, case.get("message_options", {}))

    fixture = copy.deepcopy(case)
    fixture.update(expected)
    fixture.update(errors)
    return fixture


def main() -> None:
    _bootstrap_python_baseline()

    if FIXTURE_DIR.exists():
        shutil.rmtree(FIXTURE_DIR)
    FIXTURE_DIR.mkdir(parents=True, exist_ok=True)

    cases = _build_cases()
    for case in cases:
        fixture = _run_case(case)
        out_path = FIXTURE_DIR / f"{case['case_id']}.json"
        out_path.write_text(
            json.dumps(fixture, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
            encoding="utf-8",
        )

    print(f"generated {len(cases)} fixtures into {FIXTURE_DIR}")


if __name__ == "__main__":
    main()
