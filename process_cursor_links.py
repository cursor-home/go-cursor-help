import csv
from dataclasses import dataclass
from typing import List
import json

@dataclass
class CursorVersion:
    """Cursor版本信息数据类
    用于存储Cursor的版本号和构建ID，并提供下载链接生成功能
    """
    version: str  # 版本号
    build_id: str  # 构建ID
    
    def get_download_links(self) -> dict:
        """生成所有平台的下载链接
        根据构建ID生成Windows、macOS和Linux各版本的下载链接
        
        Returns:
            dict: 包含所有平台和架构的下载链接字典
        """
        base_url = f"https://downloader.cursor.sh/builds/{self.build_id}"
        return {
            "windows": {
                "x64": f"{base_url}/windows/nsis/x64",  # Windows 64位版本
                "arm64": f"{base_url}/windows/nsis/arm64"  # Windows ARM64版本
            },
            "mac": {
                "universal": f"{base_url}/mac/installer/universal",  # macOS通用版本
                "arm64": f"{base_url}/mac/installer/arm64",  # macOS ARM64版本
                "x64": f"{base_url}/mac/installer/x64"  # macOS Intel版本
            },
            "linux": {
                "x64": f"{base_url}/linux/appImage/x64"  # Linux 64位版本
            }
        }

def parse_versions(data: str) -> List[CursorVersion]:
    """解析版本数据字符串
    将包含版本号和构建ID的文本数据解析为CursorVersion对象列表
    
    Args:
        data (str): 包含版本信息的文本数据，每行格式为"版本号,构建ID"
    
    Returns:
        List[CursorVersion]: CursorVersion对象列表
    """
    # 数据样例:
    # 输入数据格式:
    # 0.45.11,250207y6nbaw5qc
    # 0.45.10,250205buadkzpea
    # 0.45.9,250202tgstl42dt
    #
    # 解析后的对象:
    # [
    #   CursorVersion(version="0.45.11", build_id="250207y6nbaw5qc"),
    #   CursorVersion(version="0.45.10", build_id="250205buadkzpea"),
    #   CursorVersion(version="0.45.9", build_id="250202tgstl42dt")
    # ]
    versions = []
    for line in data.strip().split('\n'):
        if not line:  # 跳过空行
            continue
        version, build_id = line.strip().split(',')  # 分割每行数据
        versions.append(CursorVersion(version, build_id))
    return versions

def generate_markdown(versions: List[CursorVersion]) -> str:
    """生成Markdown格式的下载链接文档
    为每个平台和架构生成带有折叠面板的Markdown表格
    
    Args:
        versions (List[CursorVersion]): CursorVersion对象列表
    
    Returns:
        str: 生成的Markdown文档内容
    """
    # 初始化Markdown内容，包含Windows x64部分
    md = """# 🖥️ Windows

## x64
<details>
<summary style="font-size:1.2em">📦 Windows x64 安装包</summary>

| 版本 | 下载链接 |
|------|----------|
"""
    
    # 添加Windows x64版本的下载链接
    for version in versions:
        links = version.get_download_links()
        md += f"| {version.version} | [下载]({links['windows']['x64']}) |\n"
    
    # 添加Windows ARM64部分
    md += """
</details>

## ARM64 
<details>
<summary style="font-size:1.2em">📱 Windows ARM64 安装包</summary>

| 版本 | 下载链接 |
|------|----------|
"""
    
    # 添加Windows ARM64版本的下载链接
    for version in versions:
        links = version.get_download_links()
        md += f"| {version.version} | [下载]({links['windows']['arm64']}) |\n"
    
    # 添加macOS Universal部分
    md += """
</details>

# 🍎 macOS

## Universal
<details>
<summary style="font-size:1.2em">🎯 macOS Universal 安装包</summary>

| 版本 | 下载链接 |
|------|----------|
"""
    
    # 添加macOS Universal版本的下载链接
    for version in versions:
        links = version.get_download_links()
        md += f"| {version.version} | [下载]({links['mac']['universal']}) |\n"
    
    # 添加macOS ARM64部分
    md += """
</details>

## ARM64
<details>
<summary style="font-size:1.2em">💪 macOS ARM64 安装包</summary>

| 版本 | 下载链接 |
|------|----------|
"""
    
    # 添加macOS ARM64版本的下载链接
    for version in versions:
        links = version.get_download_links()
        md += f"| {version.version} | [下载]({links['mac']['arm64']}) |\n"
    
    # 添加macOS Intel部分
    md += """
</details>

## Intel
<details>
<summary style="font-size:1.2em">💻 macOS Intel 安装包</summary>

| 版本 | 下载链接 |
|------|----------|
"""
    
    # 添加macOS Intel版本的下载链接
    for version in versions:
        links = version.get_download_links()
        md += f"| {version.version} | [下载]({links['mac']['x64']}) |\n"
    
    # 添加Linux x64部分
    md += """
</details>

# 🐧 Linux

## x64
<details>
<summary style="font-size:1.2em">🎮 Linux x64 AppImage</summary>

| 版本 | 下载链接 |
|------|----------|
"""
    
    # 添加Linux x64版本的下载链接
    for version in versions:
        links = version.get_download_links()
        md += f"| {version.version} | [下载]({links['linux']['x64']}) |\n"
    
    # 添加CSS样式
    md += """
</details>

<style>
details {
    margin: 1em 0;
    padding: 0.5em 1em;
    background: #f8f9fa;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

summary {
    cursor: pointer;
    font-weight: bold;
    margin: -0.5em -1em;
    padding: 0.5em 1em;
}

summary:hover {
    background: #f1f3f5;
}

table {
    width: 100%;
    border-collapse: collapse;
    margin-top: 1em;
}

th, td {
    padding: 0.5em;
    text-align: left;
    border-bottom: 1px solid #dee2e6;
}

tr:hover {
    background: #f1f3f5;
}

a {
    color: #0366d6;
    text-decoration: none;
}

a:hover {
    text-decoration: underline;
}
</style>
"""
    return md

def main():
    """主函数
    处理版本数据并生成多种格式的输出文件
    """
    # 示例数据：包含版本号和构建ID的文本
    data = """
0.45.11,250207y6nbaw5qc
0.45.10,250205buadkzpea
# ... 更多版本数据 ...
"""
    
    # 解析版本数据
    versions = parse_versions(data)
    
    # 生成Markdown文件
    markdown_content = generate_markdown(versions)
    with open('Cursor历史.md', 'w', encoding='utf-8') as f:
        f.write(markdown_content)
    
    # 创建JSON格式的结果数据结构
    result = {
        "versions": []
    }
    
    # 处理每个版本，生成JSON数据
    for version in versions:
        version_info = {
            "version": version.version,
            "build_id": version.build_id,
            "downloads": version.get_download_links()
        }
        result["versions"].append(version_info)
    
    # 保存为JSON文件
    with open('cursor_downloads.json', 'w', encoding='utf-8') as f:
        json.dump(result, f, indent=2, ensure_ascii=False)
    
    # 生成CSV格式的下载链接
    with open('cursor_downloads.csv', 'w', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        # 写入CSV表头
        writer.writerow(['Version', 'Platform', 'Architecture', 'Download URL'])
        
        # 写入每个版本的下载链接
        for version in versions:
            links = version.get_download_links()
            for platform, archs in links.items():
                for arch, url in archs.items():
                    writer.writerow([version.version, platform, arch, url])

if __name__ == "__main__":
    main()  # 执行主函数 