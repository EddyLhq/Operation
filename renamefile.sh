#!/bin/bash
text_file="result.txt" #文件名内容
directory_path="/root/brazil/bigimages"

# 检查文件和目录是否存在
if [ ! -f "$text_file" ] || [ ! -d "$directory_path" ]; then
    echo "File or directory not found."
    exit 1
fi

# 遍历目录下的文件
cd "$directory_path" || exit
for file in *; do
    if [ -f "$file" ]; then
        if IFS= read -r new_name <&3; then
            new_filename="${new_name}.jpg"
            echo "Renaming $file to $new_filename"
            mv "$file" "$new_filename"
        else
            echo "No more names in $text_file. Exiting."
            break
        fi
    fi
done 3< "$text_file"
