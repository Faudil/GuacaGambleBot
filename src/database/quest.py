import json
from typing import List, Optional, Dict, Any
from src.database.db_handler import get_connection

def get_user_quests(user_id: int, status: Optional[str] = None) -> List[Dict[str, Any]]:
    """Retrieve all quests for a user, optionally filtered by status."""
    conn = get_connection()
    try:
        if status:
            rows = conn.execute(
                "SELECT * FROM user_quests WHERE user_id = ? AND status = ?", 
                (user_id, status)
            ).fetchall()
        else:
            rows = conn.execute(
                "SELECT * FROM user_quests WHERE user_id = ?", 
                (user_id,)
            ).fetchall()
        return [dict(row) for row in rows]
    finally:
        conn.close()

def get_quest_progress(user_id: int, quest_id: str) -> Optional[Dict[str, Any]]:
    """Retrieve current progress data for a specific quest."""
    conn = get_connection()
    try:
        row = conn.execute(
            "SELECT * FROM user_quest_data WHERE user_id = ? AND quest_id = ?",
            (user_id, quest_id)
        ).fetchone()
        if row:
            data = dict(row)
            data['custom_data'] = json.loads(data['custom_data'])
            return data
        return None
    finally:
        conn.close()

def start_quest(user_id: int, quest_id: str, custom_data: Optional[Dict[str, Any]] = None):
    """Initialize a quest for a user."""
    conn = get_connection()
    try:
        conn.execute(
            "INSERT OR REPLACE INTO user_quests (user_id, quest_id, status, started_at) VALUES (?, ?, 'ACTIVE', CURRENT_TIMESTAMP)",
            (user_id, quest_id)
        )
        c_data_json = json.dumps(custom_data) if custom_data else '{}'
        conn.execute(
            "INSERT OR REPLACE INTO user_quest_data (user_id, quest_id, step_index, progress_value, custom_data) VALUES (?, ?, 0, 0, ?)",
            (user_id, quest_id, c_data_json)
        )
        conn.commit()
    finally:
        conn.close()

def delete_quest(user_id: int, quest_id: str):
    """Remove a quest and its data for a user."""
    conn = get_connection()
    try:
        conn.execute("DELETE FROM user_quests WHERE user_id = ? AND quest_id = ?", (user_id, quest_id))
        # user_quest_data should be deleted by CASCADE, but let's be safe
        conn.execute("DELETE FROM user_quest_data WHERE user_id = ? AND quest_id = ?", (user_id, quest_id))
        conn.commit()
    finally:
        conn.close()

def update_quest_progress(
    user_id: int, 
    quest_id: str, 
    step_index: Optional[int] = None, 
    progress_value: Optional[int] = None, 
    custom_data: Optional[Dict[str, Any]] = None
):
    """Update progress for an active quest."""
    conn = get_connection()
    try:
        updates = []
        params = []
        if step_index is not None:
            updates.append("step_index = ?")
            params.append(step_index)
        if progress_value is not None:
            updates.append("progress_value = ?")
            params.append(progress_value)
        if custom_data is not None:
            updates.append("custom_data = ?")
            params.append(json.dumps(custom_data))
        
        if not updates:
            return

        params.extend([user_id, quest_id])
        query = f"UPDATE user_quest_data SET {', '.join(updates)} WHERE user_id = ? AND quest_id = ?"
        conn.execute(query, params)
        conn.commit()
    finally:
        conn.close()

def complete_quest(user_id: int, quest_id: str):
    """Mark a quest as completed."""
    conn = get_connection()
    try:
        conn.execute(
            "UPDATE user_quests SET status = 'COMPLETED', completed_at = CURRENT_TIMESTAMP WHERE user_id = ? AND quest_id = ?",
            (user_id, quest_id)
        )
        conn.commit()
    finally:
        conn.close()

def is_quest_completed(user_id: int, quest_id: str) -> bool:
    """Check if a user has completed a specific quest."""
    conn = get_connection()
    try:
        row = conn.execute(
            "SELECT 1 FROM user_quests WHERE user_id = ? AND quest_id = ? AND status = 'COMPLETED'",
            (user_id, quest_id)
        ).fetchone()
        return row is not None
    finally:
        conn.close()

def has_daily_quest_today(user_id: int) -> bool:
    """Check if the user has already received or completed a daily quest today."""
    conn = get_connection()
    try:
        row = conn.execute(
            "SELECT 1 FROM user_quests WHERE user_id = ? AND quest_id = 'daily_quest' AND date(started_at, 'localtime') = date('now', 'localtime')",
            (user_id,)
        ).fetchone()
        return row is not None
    finally:
        conn.close()

