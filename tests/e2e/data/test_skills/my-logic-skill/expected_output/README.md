# Expected Output Files

This directory contains expected output files for the logic skill tests.

## File Structure

- `skill_info.json` - Expected output from `skill-hub skill info` command
- `skill_list.json` - Expected output from `skill-hub skill list` command  
- `skill_validate.json` - Expected output from `skill-hub skill validate` command
- `skill_apply/` - Directory for skill apply operation outputs
- `skill_update/` - Directory for skill update operation outputs

## Usage

These files are used by the FileValidator class to verify that skill-hub commands produce the expected output.

## File Formats

All JSON files should contain the exact expected structure that skill-hub commands produce. The FileValidator performs exact matching against these files.

## Updates

When skill-hub output changes, update these files to match the new expected output.