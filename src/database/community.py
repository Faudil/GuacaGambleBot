import json
from sqlite3 import IntegrityError
from src.database.db_handler import get_connection

def get_server_projects(server_id: int) -> dict:
    conn = get_connection()
    c = conn.cursor()
    c.execute("SELECT project_id, level FROM server_projects WHERE server_id = ?", (server_id,))
    projects = {row["project_id"]: row["level"] for row in c.fetchall()}
    conn.close()
    return projects

def get_project_level(server_id: int, project_id: str) -> int:
    conn = get_connection()
    c = conn.cursor()
    c.execute("SELECT level FROM server_projects WHERE server_id = ? AND project_id = ?", (server_id, project_id))
    row = c.fetchone()
    conn.close()
    if row:
        return row["level"]
    return 1

def get_project_contributions(server_id: int, project_id: str) -> dict:
    conn = get_connection()
    c = conn.cursor()
    c.execute("SELECT resource_type, amount_contributed FROM server_project_contributions WHERE server_id = ? AND project_id = ?", (server_id, project_id))
    contributions = {row["resource_type"]: row["amount_contributed"] for row in c.fetchall()}
    conn.close()
    return contributions

def add_project_contribution(server_id: int, project_id: str, resource_type: str, amount: int) -> None:
    conn = get_connection()
    c = conn.cursor()
    c.execute("""
        INSERT INTO server_project_contributions (server_id, project_id, resource_type, amount_contributed)
        VALUES (?, ?, ?, ?)
        ON CONFLICT(server_id, project_id, resource_type) 
        DO UPDATE SET amount_contributed = amount_contributed + excluded.amount_contributed
    """, (server_id, project_id, resource_type, amount))
    conn.commit()
    conn.close()

def set_project_level(server_id: int, project_id: str, level: int) -> None:
    conn = get_connection()
    c = conn.cursor()
    c.execute("""
        INSERT INTO server_projects (server_id, project_id, level)
        VALUES (?, ?, ?)
        ON CONFLICT(server_id, project_id) 
        DO UPDATE SET level = excluded.level
    """, (server_id, project_id, level))
    conn.commit()
    conn.close()

def reset_project_contributions(server_id: int, project_id: str) -> None:
    conn = get_connection()
    c = conn.cursor()
    c.execute("DELETE FROM server_project_contributions WHERE server_id = ? AND project_id = ?", (server_id, project_id))
    conn.commit()
    conn.close()

def get_user_community_stats(user_id: int, server_id: int) -> dict:
    conn = get_connection()
    c = conn.cursor()
    c.execute("SELECT total_money_invested, total_items_invested FROM user_community_stats WHERE user_id = ? AND server_id = ?", (user_id, server_id))
    row = c.fetchone()
    conn.close()
    if row:
        return {"money": row["total_money_invested"], "items": row["total_items_invested"]}
    return {"money": 0, "items": 0}

def get_total_user_community_stats(user_id: int) -> dict:
    conn = get_connection()
    c = conn.cursor()
    c.execute("SELECT SUM(total_money_invested) as money, SUM(total_items_invested) as items FROM user_community_stats WHERE user_id = ?", (user_id,))
    row = c.fetchone()
    conn.close()
    if row and row["money"] is not None:
        return {"money": row["money"], "items": row["items"]}
    return {"money": 0, "items": 0}

def add_user_community_stats(user_id: int, server_id: int, money: int = 0, items: int = 0) -> None:
    conn = get_connection()
    c = conn.cursor()
    c.execute("""
        INSERT INTO user_community_stats (user_id, server_id, total_money_invested, total_items_invested)
        VALUES (?, ?, ?, ?)
        ON CONFLICT(user_id, server_id) 
        DO UPDATE SET total_money_invested = total_money_invested + excluded.total_money_invested,
                      total_items_invested = total_items_invested + excluded.total_items_invested
    """, (user_id, server_id, money, items))
    conn.commit()
    conn.close()
