import sqlite3

DB_FILE = './data/guacabot.db'



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

    conn.execute("""CREATE TABLE IF NOT EXISTS lotto_state
                    (
                        id INTEGER PRIMARY KEY,
                        winning_number INTEGER,
                        jackpot INTEGER,
                        last_bonus_date TEXT DEFAULT ''
                    )""")

    conn.execute("""CREATE TABLE IF NOT EXISTS jobs
                    (
                        user_id  INTEGER,
                        job_name TEXT,
                        level    INTEGER DEFAULT 1,
                        xp       INTEGER DEFAULT 0,
                        PRIMARY KEY (user_id, job_name)
                    )""")

    conn.commit()
    conn.close()
    print("Base de données initialisée avec succès.")
