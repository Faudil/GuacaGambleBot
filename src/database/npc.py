from typing import List, Optional, Dict, Any
from src.database.db_handler import get_connection

def get_reputation(user_id: int, npc_id: str) -> Dict[str, Any]:
    """Retrieve user's reputation with a specific NPC."""
    conn = get_connection()
    try:
        row = conn.execute(
            "SELECT reputation, level FROM user_npc_reputation WHERE user_id = ? AND npc_id = ?",
            (user_id, npc_id)
        ).fetchone()
        if row:
            return dict(row)
        return {"reputation": 0, "level": 1}
    finally:
        conn.close()

def get_daily_rep(user_id: int, npc_id: str) -> int:
    """Retrieve the amount of reputation earned today."""
    from datetime import date
    date_str = date.today().isoformat()
    conn = get_connection()
    try:
        row = conn.execute(
            "SELECT amount FROM user_npc_daily_rep WHERE user_id = ? AND npc_id = ? AND date_str = ?",
            (user_id, npc_id, date_str)
        ).fetchone()
        return row["amount"] if row else 0
    finally:
        conn.close()

def rank_up_npc(user_id: int, npc_id: str):
    """Increase the level of the NPC by 1 and reset reputation points to 0."""
    conn = get_connection()
    try:
        conn.execute(
            "UPDATE user_npc_reputation SET level = level + 1, reputation = 0 WHERE user_id = ? AND npc_id = ?",
            (user_id, npc_id)
        )
        conn.commit()
    finally:
        conn.close()

def add_reputation(user_id: int, npc_id: str, amount: int) -> int:
    """Add reputation points bounded by daily limit and current rank threshold.
       Returns the actual amount of points added.
    """
    from datetime import date
    from src.models.NPC import NPCRegistry
    
    npc_model = NPCRegistry.get_npc(npc_id)
    daily_cap = npc_model.get_daily_rep_cap() if npc_model else 500
    date_str = date.today().isoformat()
    
    conn = get_connection()
    try:
        # Get current daily amount
        daily_row = conn.execute(
            "SELECT amount FROM user_npc_daily_rep WHERE user_id = ? AND npc_id = ? AND date_str = ?",
            (user_id, npc_id, date_str)
        ).fetchone()
        
        current_daily = daily_row["amount"] if daily_row else 0
        
        # Calculate how much we CAN add today
        can_add = min(amount, daily_cap - current_daily)
        if can_add <= 0:
            return 0
            
        # Get current reputation and level
        row = conn.execute(
            "SELECT reputation, level FROM user_npc_reputation WHERE user_id = ? AND npc_id = ?",
            (user_id, npc_id)
        ).fetchone()

        if not row:
            reputation = 0
            level = 1
        else:
            reputation = row["reputation"]
            level = row["level"]

        # Limit by rank threshold (100 * level)
        max_rep_for_level = 100 * level
        actual_add = min(can_add, max_rep_for_level - reputation)
        
        if actual_add <= 0:
            return 0

        new_reputation = reputation + actual_add

        if not row:
            conn.execute(
                "INSERT INTO user_npc_reputation (user_id, npc_id, reputation, level) VALUES (?, ?, ?, ?)",
                (user_id, npc_id, new_reputation, level)
            )
        else:
            conn.execute(
                "UPDATE user_npc_reputation SET reputation = ? WHERE user_id = ? AND npc_id = ?",
                (new_reputation, user_id, npc_id)
            )
            
        # Update daily rep
        if not daily_row:
            conn.execute(
                "INSERT INTO user_npc_daily_rep (user_id, npc_id, date_str, amount) VALUES (?, ?, ?, ?)",
                (user_id, npc_id, date_str, actual_add)
            )
        else:
            conn.execute(
                "UPDATE user_npc_daily_rep SET amount = amount + ? WHERE user_id = ? AND npc_id = ? AND date_str = ?",
                (actual_add, user_id, npc_id, date_str)
            )
            
        conn.commit()
        return actual_add
    finally:
        conn.close()

def get_all_npc_reputations(user_id: int) -> List[Dict[str, Any]]:
    """Retrieve all reputations for a user."""
    conn = get_connection()
    try:
        rows = conn.execute(
            "SELECT * FROM user_npc_reputation WHERE user_id = ?",
            (user_id,)
        ).fetchall()
        return [dict(row) for row in rows]
    finally:
        conn.close()
