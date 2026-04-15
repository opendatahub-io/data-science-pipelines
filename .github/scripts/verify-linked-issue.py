#!/usr/bin/env python3
import os
import re
import sys

from github import Github

def main() -> int:
    # Retrieve env vars provided by GH Action workflow
    token = os.getenv("GITHUB_TOKEN")
    pr_num = int(os.getenv("PR_NUMBER"))

    g = Github(token)
    #todo: what is the func expecting for input formatting?
    repo = g.get_repo("kubeflow/pipelines")
    pr = repo.get_pull(pr_num)

    # 1. Parse PR body for linked issues (e.g., #123 or Fixes #123)
    pr_body = pr.body or ""
    issue_numbers = re.findall(r"(?:#|issues\/)(\d+)", pr_body)

    if not issue_numbers:
        print('ERROR: No linked issues found in the PR description.')
        return 1

    #2: Check each linked issue for the "ready" command.
    # If there is more than one issue linked, each issue must be marked /ready
    found_ready = False
    for issue_num in issue_numbers:
        #todo: case in which issue_num cannot be converted into an int?
        issue = repo.get_issue(int(issue_num))
        comments = issue.get_comments()

        # Check if the issue contains a comment with /ready command
        for comment in comments:
            if "/ready" in comment.body:
                print(f"Found '/ready' command in issue #{issue_num}")
                found_ready = True
                break
        #todo: is there ever a case in which "/ready" is canceled out by a later comment? (ie do not exit here)
        if found_ready: break

    if not found_ready:
        print("ERROR: The linked issue(s) must have a '/ready' command.")
        return 1

    #todo: add successful return message.
    print()
    return 0

if __name__ == "__main__":
    sys.exit(main())


