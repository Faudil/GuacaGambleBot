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

    conn.execute("""CREATE TABLE IF NOT EXISTS user_pets (
                                                             id          INTEGER PRIMARY KEY AUTOINCREMENT,
                                                             user_id     INTEGER,
                                                             pet_type    TEXT,
                                                             nickname    TEXT,

                                                             level       INTEGER DEFAULT 1,
                                                             food_eaten  INTEGER DEFAULT 0,
                                                             bonus       INTEGER DEFAULT 0,
                                                             xp          INTEGER DEFAULT 0,

                                                             max_hp      INTEGER DEFAULT 50,
                                                             hp          INTEGER DEFAULT 50,
                                                             atk         INTEGER DEFAULT 10,
                                                             defense     INTEGER DEFAULT 5,
                                                             speed       INTEGER DEFAULT 10,
                                                             dge         INTEGER DEFAULT 5,   -- Dodge (%)
                                                             acc         INTEGER DEFAULT 0,   -- Accuracy (%)
                                                             crit_c      INTEGER DEFAULT 5,   -- Crit Chance (%)
                                                             crit_d      REAL DEFAULT 1.5,    -- Crit Damage (factor)
        
                                                             spc_c    INTEGER DEFAULT 0,   -- Special effect Chance (%)
                                                             trs_lvl     INTEGER DEFAULT 0,

                                                             elo         INTEGER DEFAULT 1000,-- Points de classement
                                                             is_active   BOOLEAN DEFAULT 0
                    )""")

    conn.execute("""
                 CREATE TABLE IF NOT EXISTS user_stats (
                                                           user_id INTEGER PRIMARY KEY,
                                                           pvp_wins INTEGER DEFAULT 0,
                                                           pvp_losses INTEGER DEFAULT 0,
                                                           pve_wins INTEGER DEFAULT 0,
                                                           items_mined INTEGER DEFAULT 0,
                                                           items_fished INTEGER DEFAULT 0,
                                                           items_farmed INTEGER DEFAULT 0,
                                                           money_earned INTEGER DEFAULT 0,
                                                           pets_fed INTEGER DEFAULT 0,


                                                           coinflip_lost INTEGER DEFAULT 0,
                                                           coinflip_won INTEGER DEFAULT 0,
                     
                                                           casino_lost INTEGER DEFAULT 0,
                                                           casino_won INTEGER DEFAULT 0,

                                                           slots_won INTEGER DEFAULT 0,
                                                           slots_lost INTEGER DEFAULT 0,
                                                           blackjack_won INTEGER DEFAULT 0,
                                                           blackjack_lost INTEGER DEFAULT 0,
                                                           roulette_won INTEGER DEFAULT 0,
                                                           roulette_lost INTEGER DEFAULT 0,
                                                           lotto_participations INTEGER DEFAULT 0,
                                                           lotto_won INTEGER DEFAULT 0,
                                                           bets_won INTEGER DEFAULT 0,
                                                           bets_lost INTEGER DEFAULT 0,
                                                           wagers_won INTEGER DEFAULT 0,
                                                           wagers_lost INTEGER DEFAULT 0,
                                                           casino_spent INTEGER DEFAULT 0,
                                                           slots_spent INTEGER DEFAULT 0,
                                                           slots_money_won INTEGER DEFAULT 0,
                                                           slots_money_lost INTEGER DEFAULT 0,
                                                           coinflip_spent INTEGER DEFAULT 0,
                                                           coinflip_money_won INTEGER DEFAULT 0,
                                                           coinflip_money_lost INTEGER DEFAULT 0,
                                                           blackjack_spent INTEGER DEFAULT 0,
                                                           blackjack_money_won INTEGER DEFAULT 0,
                                                           blackjack_money_lost INTEGER DEFAULT 0,
                                                           roulette_spent INTEGER DEFAULT 0,
                                                           roulette_money_won INTEGER DEFAULT 0,
                                                           roulette_money_lost INTEGER DEFAULT 0,
                                                           daily_uses INTEGER DEFAULT 0
                                            
                 );""")

    conn.execute("""
                 CREATE TABLE IF NOT EXISTS user_achievements (
                     user_id INTEGER,
                     achievement_id TEXT,
                     PRIMARY KEY (user_id, achievement_id),
                     FOREIGN KEY (user_id) REFERENCES users(user_id)
                 );""")

    conn.commit()
    conn.close()
    print("Base de données initialisée avec succès.")
