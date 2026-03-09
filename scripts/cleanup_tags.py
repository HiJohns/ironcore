#!/usr/bin/env python3
"""
Tag Cleanup and Aggregation Script
标签聚合与清洗脚本

功能：
1. 合并同义词（如 ai-agent, aiagent, agent → ai-agent）
2. 修复断词（如 open, source → open-source）
3. 过滤低频词（仅出现1次且非监控关键词）
4. 合并破碎标签（如 ai + agent → ai-agent）
"""

import os
import sys
import sqlite3
import re
import yaml
from typing import List, Dict, Set, Tuple
from collections import defaultdict

DB_PATH = os.path.join(os.path.dirname(__file__), '..', 'ironcore.db')
CONFIG_PATH = os.path.join(os.path.dirname(__file__), '..', 'config.yaml')


class TagCleaner:
    """标签清洗器"""
    
    def __init__(self, db_path: str = DB_PATH, config_path: str = CONFIG_PATH):
        self.db_path = db_path
        self.config_path = config_path
        self.conn = None
        self.cursor = None
        self.protected_tags = set()  # 监控关键词，不能删除
        self.synonym_map = {}  # 同义词映射
        self.broken_word_patterns = []  # 断词修复模式
        
    def load_config(self):
        """加载配置文件，获取监控关键词"""
        try:
            with open(self.config_path, 'r', encoding='utf-8') as f:
                config = yaml.safe_load(f)
            
            # 从 sentinel_keywords 获取监控关键词
            sentinel_keywords = config.get('sentinel_keywords', {})
            for category, keywords in sentinel_keywords.items():
                for kw in keywords:
                    # 标准化关键词（转小写，替换空格为连字符）
                    normalized = kw.lower().replace(' ', '-')
                    self.protected_tags.add(normalized)
                    # 也保护原始形式
                    self.protected_tags.add(kw.lower())
            
            print(f"[INFO] 加载了 {len(self.protected_tags)} 个监控关键词")
        except Exception as e:
            print(f"[WARN] 加载配置失败: {type(e).__name__}")
            # 默认保护一些核心关键词
            self.protected_tags = {
                'hormuz', 'strait', 'sanctions', 'embargo',
                'ga2o3', 'gallium-oxide', 'transformer-shortage', 'power-grid',
                'fed-rate', 'inflation', 'recession', 'fed', 'vix', 'transformer'
            }
    
    def connect_db(self):
        """连接数据库"""
        try:
            self.conn = sqlite3.connect(self.db_path)
            self.conn.row_factory = sqlite3.Row
            self.cursor = self.conn.cursor()
            print("[INFO] 数据库连接成功")
        except sqlite3.Error as e:
            raise RuntimeError(f"[DB] Connection failed: {e}")
    
    def close_db(self):
        """关闭数据库连接"""
        if self.conn:
            self.conn.close()
            print("[INFO] 数据库连接已关闭")
    
    def get_all_tags(self) -> Dict[str, int]:
        """获取所有标签及其使用次数"""
        self.cursor.execute('''
            SELECT t.name, COUNT(it.item_id) as count
            FROM tags t
            LEFT JOIN item_tags it ON t.id = it.tag_id
            GROUP BY t.id, t.name
            ORDER BY count DESC
        ''')
        return {row['name']: row['count'] for row in self.cursor.fetchall()}
    
    def build_synonym_map(self, tags: Dict[str, int]) -> Dict[str, str]:
        """构建同义词映射"""
        synonym_groups = [
            # AI 相关
            ['ai-agent', 'aiagent', 'agent', 'ai-agents', 'agents'],
            # 开源相关
            ['open-source', 'opensource', 'open source'],
            # 电力相关
            ['power-grid', 'powergrid', 'power grid', 'grid'],
            # 变压器相关
            ['transformer-shortage', 'transformershortage', 'transformer shortage', 'transformer'],
            # 地缘政治
            ['hormuz-strait', 'hormuz', 'strait-of-hormuz'],
            ['gallium-oxide', 'ga2o3', 'gallium oxide'],
        ]
        
        synonym_map = {}
        for group in synonym_groups:
            # 选择最常用的形式作为标准形式
            valid_tags = [(tag, tags.get(tag, 0)) for tag in group if tag in tags]
            if valid_tags:
                # 选择出现次数最多的作为标准形式
                canonical = max(valid_tags, key=lambda x: x[1])[0]
                for tag in group:
                    if tag in tags and tag != canonical:
                        synonym_map[tag] = canonical
        
        return synonym_map
    
    def find_broken_words(self, tags: Dict[str, int]) -> List[Tuple[str, str]]:
        """查找需要修复的断词"""
        broken_pairs = []
        tag_list = list(tags.keys())
        
        # 常见的断词模式
        for i, tag1 in enumerate(tag_list):
            for tag2 in tag_list[i+1:]:
                # 检查是否是常见的拆分模式
                combined = f"{tag1}-{tag2}"
                reverse_combined = f"{tag2}-{tag1}"
                
                # 如果组合形式存在且比分开的更常用，则记录
                if combined in tags:
                    if tags[combined] >= tags[tag1] and tags[combined] >= tags[tag2]:
                        broken_pairs.append((tag1, tag2, combined))
                elif reverse_combined in tags:
                    if tags[reverse_combined] >= tags[tag1] and tags[reverse_combined] >= tags[tag2]:
                        broken_pairs.append((tag2, tag1, reverse_combined))
        
        return broken_pairs
    
    def merge_tags(self, from_tag: str, to_tag: str):
        """合并两个标签（将所有关联从 from_tag 移动到 to_tag）"""
        try:
            # 获取标签 ID
            self.cursor.execute('SELECT id FROM tags WHERE name = ?', (from_tag,))
            from_row = self.cursor.fetchone()
            if not from_row:
                return
            from_id = from_row['id']
            
            self.cursor.execute('SELECT id FROM tags WHERE name = ?', (to_tag,))
            to_row = self.cursor.fetchone()
            if not to_row:
                # 目标标签不存在，重命名源标签
                self.cursor.execute('UPDATE tags SET name = ? WHERE id = ?', (to_tag, from_id))
                print(f"[MERGE] 重命名标签: '{from_tag}' → '{to_tag}'")
            else:
                to_id = to_row['id']
                # 移动关联关系
                self.cursor.execute('''
                    UPDATE OR IGNORE item_tags 
                    SET tag_id = ? 
                    WHERE tag_id = ?
                ''', (to_id, from_id))
                moved_count = self.cursor.rowcount
                # 删除源标签
                self.cursor.execute('DELETE FROM tags WHERE id = ?', (from_id,))
                print(f"[MERGE] 合并标签: '{from_tag}' → '{to_tag}' (移动了 {moved_count} 个关联)")
            
            self.conn.commit()
        except Exception as e:
            print(f"[ERROR] 合并标签失败 '{from_tag}' → '{to_tag}': {e}")
            self.conn.rollback()
    
    def delete_low_frequency_tags(self, tags: Dict[str, int], min_count: int = 2):
        """删除低频标签"""
        deleted = 0
        for tag, count in tags.items():
            if count < min_count and tag.lower() not in self.protected_tags:
                try:
                    self.cursor.execute('SELECT id FROM tags WHERE name = ?', (tag,))
                    row = self.cursor.fetchone()
                    if row:
                        tag_id = row['id']
                        # 先删除关联
                        self.cursor.execute('DELETE FROM item_tags WHERE tag_id = ?', (tag_id,))
                        # 再删除标签
                        self.cursor.execute('DELETE FROM tags WHERE id = ?', (tag_id,))
                        print(f"[DELETE] 删除低频标签: '{tag}' (出现 {count} 次)")
                        deleted += 1
                except Exception as e:
                    print(f"[ERROR] 删除标签失败 '{tag}': {e}")
        
        self.conn.commit()
        print(f"[INFO] 共删除 {deleted} 个低频标签")
        return deleted
    
    def cleanup(self):
        """执行完整的清理流程"""
        print("\n" + "="*60)
        print("开始标签清洗流程")
        print("="*60 + "\n")
        
        # 1. 加载配置
        self.load_config()
        
        # 2. 连接数据库
        self.connect_db()
        
        # 3. 获取所有标签
        tags = self.get_all_tags()
        print(f"[INFO] 数据库中共有 {len(tags)} 个标签")
        print(f"[INFO] 其中 {len(self.protected_tags)} 个监控关键词受保护")
        
        # 4. 合并同义词
        print("\n[STEP 1] 合并同义词...")
        synonym_map = self.build_synonym_map(tags)
        if synonym_map:
            print(f"[INFO] 发现 {len(synonym_map)} 组同义词需要合并")
            for from_tag, to_tag in synonym_map.items():
                self.merge_tags(from_tag, to_tag)
        else:
            print("[INFO] 没有发现需要合并的同义词")
        
        # 5. 刷新标签列表
        tags = self.get_all_tags()
        
        # 6. 修复断词
        print("\n[STEP 2] 修复断词...")
        broken_pairs = self.find_broken_words(tags)
        if broken_pairs:
            print(f"[INFO] 发现 {len(broken_pairs)} 组断词需要修复")
            for tag1, tag2, combined in broken_pairs:
                print(f"[INFO] 建议: '{tag1}' + '{tag2}' → '{combined}'")
                # 自动合并
                self.merge_tags(tag1, combined)
                self.merge_tags(tag2, combined)
        else:
            print("[INFO] 没有发现需要修复的断词")
        
        # 7. 刷新标签列表
        tags = self.get_all_tags()
        
        # 8. 删除低频标签
        print("\n[STEP 3] 删除低频标签（出现1次且非监控关键词）...")
        deleted_count = self.delete_low_frequency_tags(tags, min_count=2)
        
        # 9. 统计信息
        print("\n" + "="*60)
        print("清理完成")
        print("="*60)
        final_tags = self.get_all_tags()
        print(f"[SUMMARY] 清理前: {len(tags)} 个标签")
        print(f"[SUMMARY] 清理后: {len(final_tags)} 个标签")
        print(f"[SUMMARY] 减少: {len(tags) - len(final_tags)} 个标签")
        
        # 10. 关闭连接
        self.close_db()


def main():
    """主函数"""
    cleaner = TagCleaner()
    try:
        cleaner.cleanup()
    except KeyboardInterrupt:
        print("\n[INTERRUPT] 用户中断")
        cleaner.close_db()
        sys.exit(1)
    except Exception as e:
        print(f"\n[ERROR] 执行失败: {e}")
        cleaner.close_db()
        sys.exit(1)


if __name__ == '__main__':
    main()