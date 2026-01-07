#!/usr/bin/env python3
import sys
import json
import re
import os

def main():
    try:
        # Read input from stdin
        input_data = sys.stdin.read()
        data = json.loads(input_data)
        prompt = data.get('prompt', '').lower()

        # Load skill rules
        cwd = os.getcwd()
        project_dir = os.environ.get('CLAUDE_PROJECT_DIR')
        if not project_dir:
            if cwd.endswith('.claude/hooks'):
                project_dir = os.path.dirname(os.path.dirname(cwd))
            else:
                project_dir = cwd

        rules_path = os.path.join(project_dir, '.claude', 'skills', 'skill-rules.json')

        with open(rules_path, 'r') as f:
            rules = json.load(f)

        matched_skills = []

        # Check each skill for matches
        for skill_name, config in rules.get('skills', {}).items():
            triggers = config.get('promptTriggers')
            if not triggers:
                continue

            # Keyword matching
            keywords = triggers.get('keywords', [])
            keyword_match = any(kw.lower() in prompt for kw in keywords)
            if keyword_match:
                matched_skills.append({
                    'name': skill_name,
                    'match_type': 'keyword',
                    'config': config
                })
                continue

            # Intent pattern matching
            intent_patterns = triggers.get('intentPatterns', [])
            intent_match = any(re.search(pattern, prompt, re.IGNORECASE) for pattern in intent_patterns)
            if intent_match:
                matched_skills.append({
                    'name': skill_name,
                    'match_type': 'intent',
                    'config': config
                })

        # Generate output if matches found
        if matched_skills:
            output = '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n'
            output += '🎯 SKILL ACTIVATION CHECK\n'
            output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n'

            # Group by priority
            critical = [s for s in matched_skills if s['config'].get('priority') == 'critical']
            high = [s for s in matched_skills if s['config'].get('priority') == 'high']
            medium = [s for s in matched_skills if s['config'].get('priority') == 'medium']
            low = [s for s in matched_skills if s['config'].get('priority') == 'low']

            if critical:
                output += '⚠️ CRITICAL SKILLS (REQUIRED):\n'
                for s in critical:
                    output += f"  → {s['name']}\n"
                output += '\n'

            if high:
                output += '📚 RECOMMENDED SKILLS:\n'
                for s in high:
                    output += f"  → {s['name']}\n"
                output += '\n'

            if medium:
                output += '💡 SUGGESTED SKILLS:\n'
                for s in medium:
                    output += f"  → {s['name']}\n"
                output += '\n'

            if low:
                output += '📌 OPTIONAL SKILLS:\n'
                for s in low:
                    output += f"  → {s['name']}\n"
                output += '\n'

            output += 'ACTION: Use Skill tool BEFORE responding\n'
            output += '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n'

            print(output)

        sys.exit(0)

    except Exception as e:
        print(f'Error in skill-activation-prompt hook: {e}', file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    main()
