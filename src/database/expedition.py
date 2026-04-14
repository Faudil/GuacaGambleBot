import json
from datetime import datetime
from src.database.db_handler import get_connection

def start_expedition(user_id: int, pet_id: int, duration_hours: int, reward_xp: int, reward_items: list, log: list):
    conn = get_connection()
    try:
        start_time = datetime.now().isoformat()
        end_time = (datetime.now().replace(hour=datetime.now().hour + duration_hours) if datetime.now().hour + duration_hours < 24 else datetime.now()).isoformat()
        # Better time calculation
        import datetime as dt
        end_time = (dt.datetime.now() + dt.timedelta(hours=duration_hours)).isoformat()
        
        conn.execute("""
            INSERT INTO pet_expeditions (user_id, pet_id, start_time, end_time, reward_xp, reward_items, log)
            VALUES (?, ?, ?, ?, ?, ?, ?)
        """, (user_id, pet_id, start_time, end_time, reward_xp, json.dumps(reward_items), json.dumps(log)))
        
        conn.execute("UPDATE user_pets SET on_expedition = 1 WHERE id = ?", (pet_id,))
        conn.commit()
    finally:
        conn.close()

def get_active_expedition(user_id: int):
    conn = get_connection()
    try:
        row = conn.execute("""
            SELECT * FROM pet_expeditions 
            WHERE user_id = ? AND is_claimed = 0 
            ORDER BY id DESC LIMIT 1
        """, (user_id,)).fetchone()
        return dict(row) if row else None
    finally:
        conn.close()

def get_expedition_by_id(exp_id: int):
    conn = get_connection()
    try:
        row = conn.execute("SELECT * FROM pet_expeditions WHERE id = ?", (exp_id,)).fetchone()
        return dict(row) if row else None
    finally:
        conn.close()

def claim_expedition(exp_id: int):
    conn = get_connection()
    try:
        exp = get_expedition_by_id(exp_id)
        if not exp:
            return False
            
        conn.execute("UPDATE pet_expeditions SET is_claimed = 1 WHERE id = ?", (exp_id,))
        conn.execute("UPDATE user_pets SET on_expedition = 0 WHERE id = ?", (exp['pet_id'],))
        conn.commit()
        return True
    finally:
        conn.close()

def is_pet_on_expedition(pet_id: int) -> bool:
    conn = get_connection()
    try:
        row = conn.execute("SELECT on_expedition FROM user_pets WHERE id = ?", (pet_id,)).fetchone()
        return bool(row['on_expedition']) if row else False
    finally:
        conn.close()
