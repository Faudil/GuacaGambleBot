import sqlite3
from datetime import datetime

DB_FILE = './data/guacabot.db'
STARTING_BALANCE = 100


def get_connection():
    conn = sqlite3.connect(DB_FILE)
    conn.row_factory = sqlite3.Row
    return conn


def init_db():
    conn = get_connection()
    c = conn.cursor()

    c.execute("""CREATE TABLE IF NOT EXISTS users
                 (
                     user_id INTEGER PRIMARY KEY,
                     balance INTEGER DEFAULT 100
                 )""")

    c.execute("""CREATE TABLE IF NOT EXISTS cooldowns
        (
            user_id INTEGER,
            activity_name TEXT,
            last_used TIMESTAMP,
            PRIMARY KEY(user_id, activity_name)
        )""")
    c.execute("""CREATE TABLE IF NOT EXISTS game_limits
    (
        user_id INTEGER,
        game_name TEXT,
        date_str TEXT,
        count INTEGER DEFAULT 0,
        PRIMARY KEY(user_id, game_name)
        )""")

    c.execute("""CREATE TABLE IF NOT EXISTS bets
                 (
                     id INTEGER PRIMARY KEY AUTOINCREMENT,
                     creator_id INTEGER,
                     description TEXT,
                     option1 TEXT,
                     option2 TEXT,
                     status VARCHAR(16) DEFAULT 'OPEN',
                     winner CHAR
                 )""")

    c.execute("""CREATE TABLE IF NOT EXISTS wagers
    (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        bet_id INTEGER,
        user_id INTEGER,
        option TEXT,
        amount INTEGER,
        FOREIGN KEY (bet_id) REFERENCES bets(id)
        )""")

    c.execute("""CREATE TABLE IF NOT EXISTS items
                 (
                     id INTEGER PRIMARY KEY AUTOINCREMENT,
                     name TEXT UNIQUE,
                     price INTEGER,
                     description TEXT,
                     effect_type TEXT
                 )""")

    c.execute("""CREATE TABLE IF NOT EXISTS inventory
    (
        user_id INTEGER,
        item_id INTEGER,
        quantity INTEGER DEFAULT 1,
        FOREIGN KEY (user_id) REFERENCES users
                 (user_id),
        FOREIGN KEY
                 (item_id) REFERENCES items
                 (id),
        PRIMARY KEY (user_id, item_id)
        )""")

    conn.commit()
    conn.close()
    print("Base de données initialisée avec succès.")


def get_balance(user_id):
    conn = get_connection()
    row = conn.execute("SELECT balance FROM users WHERE user_id = ?", (user_id,)).fetchone()
    conn.close()
    if row:
        return row['balance']
    update_balance(user_id, 0)
    return STARTING_BALANCE


def update_balance(user_id, amount) -> int:
    conn = get_connection()
    try:
        conn.execute("""
                     INSERT INTO users (user_id, balance)
                     VALUES (?, ?) ON CONFLICT(user_id) DO
                     UPDATE SET balance = balance + ?
                     """, (user_id, STARTING_BALANCE + amount, amount))
        conn.commit()
        new_bal = conn.execute("SELECT balance FROM users WHERE user_id = ?", (user_id,)).fetchone()['balance']
        return new_bal
    finally:
        conn.close()


def get_top_users(limit=5):
    conn = get_connection()
    rows = conn.execute("SELECT user_id, balance FROM users ORDER BY balance DESC LIMIT ?", (limit,)).fetchall()
    conn.close()
    for row in rows:
        yield {"user_id": row[0], "balance": row[1]}

def get_cooldown(user_id, activity_name):
    """Récupère la date de la dernière utilisation sous forme de string ISO."""
    conn = get_connection()
    row = conn.execute("SELECT last_used FROM cooldowns WHERE user_id = ? AND activity_name = ?",
                       (user_id, activity_name)).fetchone()
    conn.close()
    if row:
        return row['last_used']
    return None


def set_cooldown(user_id, activity_name):
    """Met à jour le cooldown à 'maintenant'."""
    now_iso = datetime.now().isoformat()
    conn = get_connection()
    conn.execute("""
                 INSERT INTO cooldowns (user_id, activity_name, last_used)
                 VALUES (?, ?, ?) ON CONFLICT(user_id, activity_name) DO
                 UPDATE SET last_used = ?
                 """, (user_id, activity_name, now_iso, now_iso))
    conn.commit()
    conn.close()


def check_game_limit(user_id, game_name, max_usage):
    """Retourne (autorisé: bool, restants: int). Gère le reset journalier auto."""
    conn = get_connection()
    today_str = datetime.now().strftime("%Y-%m-%d")
    row = conn.execute("SELECT date_str, count FROM game_limits WHERE user_id = ? AND game_name = ?",
                       (user_id, game_name)).fetchone()
    current_count = 0
    if row:
        if row['date_str'] != today_str:
            conn.execute("UPDATE game_limits SET date_str = ?, count = 0 WHERE user_id = ? AND game_name = ?",
                         (today_str, user_id, game_name))
            conn.commit()
            current_count = 0
        else:
            current_count = row['count']
    conn.close()
    if current_count >= max_usage:
        return False, 0
    return True, max_usage - current_count


def increment_game_limit(user_id, game_name):
    today_str = datetime.now().strftime("%Y-%m-%d")
    conn = get_connection()
    conn.execute("""
                 INSERT INTO game_limits (user_id, game_name, date_str, count)
                 VALUES (?, ?, ?, 1) ON CONFLICT(user_id, game_name) DO
                 UPDATE SET count = count + 1, date_str = ?
                 """, (user_id, game_name, today_str, today_str))
    conn.commit()
    conn.close()


def create_bet_db(creator_id, description, opt1, opt2):
    conn = get_connection()
    cursor = conn.cursor()
    cursor.execute("""
                   INSERT INTO bets (creator_id, description, option1, option2, status)
                   VALUES (?, ?, ?, ?, 'OPEN')
                   """, (creator_id, description, opt1, opt2))
    bet_id = cursor.lastrowid
    conn.commit()
    conn.close()
    return bet_id


def get_bet_data(bet_id):
    """Récupère tout : infos du pari ET liste des mises."""
    conn = get_connection()
    bet = conn.execute("SELECT * FROM bets WHERE id = ?", (bet_id,)).fetchone()
    if not bet:
        conn.close()
        return None
    wagers = conn.execute("SELECT * FROM wagers WHERE bet_id = ?", (bet_id,)).fetchall()
    conn.close()
    return {
        "id": bet['id'],
        "creator": bet['creator_id'],
        "description": bet['description'],
        "options": [bet['option1'], bet['option2']],
        "status": bet['status'],
        "winner": bet['winner'],
        "wagers": [{"user_id": w['user_id'], "option": w['option'], "amount": w['amount']} for w in wagers]
    }

def freeze_bet(bet_id):
    conn = get_connection()
    conn.execute("UPDATE bets SET status = 'FROZEN' WHERE id = ?", (bet_id,))
    conn.close()

def add_wager(bet_id, user_id, option, amount):
    conn = get_connection()
    conn.execute("INSERT INTO wagers (bet_id, user_id, option, amount) VALUES (?, ?, ?, ?)",
                 (bet_id, user_id, option, amount))
    conn.commit()
    conn.close()


def close_bet_db(bet_id, winner):
    conn = get_connection()
    conn.execute("UPDATE bets SET status = 'CLOSED', winner = ? WHERE id = ?", (winner, bet_id))
    conn.commit()
    conn.close()
