#!/usr/bin/env python3
"""
Script to check if integration test checkbox is checked for main -> stable merges.
Uses PyGithub SDK for GitHub API interactions.
"""

import os
import re
import sys
from github import Github


def check_integration_test_checkbox(pr_body):
    """Check if the integration test checkbox is checked in the PR body."""
    if not pr_body:
        return False

    # Pattern to match checked checkbox for integration tests
    checkbox_pattern = r'- \[x\].*integration.*tests.*openshift.*cluster.*odh.*nightly'
    return bool(re.search(checkbox_pattern, pr_body, re.IGNORECASE))


def has_unchecked_integration_test_checkbox(pr_body):
    """Check if there's an unchecked integration test checkbox in the PR body."""
    if not pr_body:
        return False

    # Pattern to match unchecked checkbox for integration tests
    unchecked_pattern = r'- \[ \].*integration.*tests.*openshift.*cluster.*odh.*nightly'
    return bool(re.search(unchecked_pattern, pr_body, re.IGNORECASE))


def remove_integration_test_checkbox(pr_body):
    """Remove any integration test checkbox (checked or unchecked) from PR body."""
    if not pr_body:
        return pr_body

    # Pattern to match any integration test checkbox (checked or unchecked)
    checkbox_pattern = r'- \[[x ]\].*integration.*tests.*openshift.*cluster.*odh.*nightly.*\n?'
    updated_body = re.sub(checkbox_pattern, '', pr_body, flags=re.IGNORECASE)
    return updated_body.strip()


def post_instruction_comment(pull_request):
    """Post instruction comment if it doesn't already exist."""
    instruction_comment = """## 🚦 Integration Test Verification Required

This pull request is merging **main** → **stable** and requires integration test verification.

### ✅ Required Action:
Please add the following checkbox to your PR description and check it **only after** running the integration tests:

```markdown
- [ ] Ran integration tests in an OpenShift cluster with latest ODH nightly
```

### 📝 Steps:
1. Run integration tests in OpenShift cluster with latest ODH nightly
2. Fetch nightly build information from **#odh-nightlies-notifications** Slack channel
3. Edit this PR description to add the checkbox above
4. Check the checkbox to confirm tests were completed
5. This workflow check will automatically pass once the checkbox is detected

---
*This requirement ensures production stability by verifying integration tests against the latest ODH nightly build.*"""

    # Check if instruction comment already exists
    comments = pull_request.get_issue_comments()
    for comment in comments:
        if ("Integration Test Verification Required" in comment.body and
            comment.user.type == "Bot"):
            print("ℹ️ Instruction comment already exists")
            return

    # Post new comment
    try:
        pull_request.create_issue_comment(instruction_comment)
        print("✅ Posted instruction comment")
    except Exception as e:
        print(f"⚠️ Failed to post comment: {e}")


def main():
    """Main function to check integration test requirement."""
    # Get environment variables
    token = os.getenv("GITHUB_TOKEN")
    pr_number = os.getenv("PR_NUMBER")
    repo_owner = os.getenv("REPO_OWNER")
    repo_name = os.getenv("REPO_NAME")
    github_event_name = os.getenv("GITHUB_EVENT_NAME")
    github_event_action = os.getenv("GITHUB_EVENT_ACTION")

    if not all([token, pr_number, repo_owner, repo_name]):
        print("❌ Missing required environment variables")
        sys.exit(1)

    try:
        pr_number = int(pr_number)
    except ValueError:
        print(f"❌ Invalid PR number: {pr_number}")
        sys.exit(1)

    print(f"🔍 Checking PR #{pr_number} in {repo_owner}/{repo_name}")
    print(f"📝 Event: {github_event_name}, Action: {github_event_action}")

    # Initialize GitHub client
    try:
        github_client = Github(token)
        repo = github_client.get_repo(f"{repo_owner}/{repo_name}")
        pull_request = repo.get_pull(pr_number)
    except Exception as e:
        print(f"❌ Error accessing GitHub API: {e}")
        sys.exit(1)

    # Get PR body
    pr_body = pull_request.body or ""

    # If this is a synchronize event (new commits), remove any existing integration test checkbox
    if github_event_action == "synchronize":
        print("🔄 New commits detected - checking for existing integration test checkbox")

        if (check_integration_test_checkbox(pr_body) or
            has_unchecked_integration_test_checkbox(pr_body)):

            print("🗑️ Removing existing integration test checkbox due to new commits")
            updated_body = remove_integration_test_checkbox(pr_body)

            try:
                pull_request.edit(body=updated_body)
                print("✅ Successfully removed integration test checkbox")

                # Post a comment explaining the removal
                removal_comment = """## 🔄 Integration Test Checkbox Removed

New commits have been pushed to this PR. The integration test checkbox has been automatically removed to ensure tests are re-run with the latest changes.

**Next Steps:**
1. Re-run integration tests in OpenShift cluster with latest ODH nightly
2. Add the integration test checkbox back to the PR description
3. Check the checkbox only after confirming tests pass with the new commits

```markdown
- [ ] Ran integration tests in an OpenShift cluster with latest ODH nightly
```"""

                pull_request.create_issue_comment(removal_comment)
                print("✅ Posted checkbox removal notification")

            except Exception as e:
                print(f"⚠️ Failed to remove checkbox from PR body: {e}")
        else:
            print("ℹ️ No existing integration test checkbox found")

    # Check for integration test checkbox
    has_checkbox = check_integration_test_checkbox(pr_body)

    if has_checkbox:
        print("✅ Integration test checkbox verified - merge can proceed")
        print("✅ Found checked integration test checkbox in PR description")
        sys.exit(0)
    else:
        print("❌ Integration test verification required for main → stable merge")

        # Post instruction comment
        post_instruction_comment(pull_request)

        print("\n📋 Required: Add the following checkbox to your PR description and check it:")
        print("- [ ] Ran integration tests in an OpenShift cluster with latest ODH nightly")
        print("\n⚠️ Important: Only check this box after actually running the integration tests!")

        sys.exit(1)


if __name__ == "__main__":
    main()