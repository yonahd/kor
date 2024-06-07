#!/bin/bash

# Find all unused resources in cluster
all_json=$(kor all --group-by=resource -o json)

# Read all top-level keys (resource types)
resource_types=$(echo $all_json | jq -r 'keys[]')

output_dir="exceptions"
if [ -d "$output_dir" ]; then
    rm -r "$output_dir"
fi

mkdir -p "$output_dir"

# Process each resource type
for resource_type in $resource_types; do
  output_file="$output_dir/${resource_type,,}s.json"

  # Format resource type exceptions
  echo $all_json | jq --arg resource_type "$resource_type" '
  {
    ("exception" + ($resource_type) + "s"): [
      .[$resource_type] | to_entries[] |
      {
        "Namespace": .key,
        "ResourceName": .value[]
      }
    ]
  }
  ' > $output_file

  echo "Processing completed for "$resource_type"s, output saved to $output_file"
done
