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
                     balance INTEGER DEFAULT 100,
                     crowns INTEGER DEFAULT 0,
                     extra_inv_slots INTEGER DEFAULT 0,
                     extra_pet_slots INTEGER DEFAULT 0,
                     boss_league_stage INTEGER DEFAULT 0
                 )""")

    # Existing tables migration
    for col, type_def in [("crowns", "INTEGER DEFAULT 0"),
                          ("extra_inv_slots", "INTEGER DEFAULT 0"),
                          ("extra_pet_slots", "INTEGER DEFAULT 0"),
                          ("boss_league_stage", "INTEGER DEFAULT 0")]:
        try:
            conn.execute(f"ALTER TABLE users ADD COLUMN {col} {type_def}")
        except sqlite3.OperationalError:
            pass

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
                                                             is_active   BOOLEAN DEFAULT 0,
                                                             on_expedition BOOLEAN DEFAULT 0
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

    conn.execute("""
                 CREATE TABLE IF NOT EXISTS server_pet_elo (
                                                               pet_id INTEGER,
                                                               server_id INTEGER,
                                                               elo INTEGER DEFAULT 1000,
                                                               PRIMARY KEY (pet_id, server_id),
                                                               FOREIGN KEY (pet_id) REFERENCES user_pets(id) ON DELETE CASCADE
                 );""")

    conn.execute("""
                 CREATE TABLE IF NOT EXISTS server_settings (
                                                                server_id INTEGER PRIMARY KEY,
                                                                announcement_channel_id INTEGER,
                                                                language TEXT DEFAULT 'fr'
                 );""")

    # Setup default column values for existing databases
    try:
        conn.execute("ALTER TABLE server_settings ADD COLUMN language TEXT DEFAULT 'fr'")
    except sqlite3.OperationalError:
        pass

    conn.execute("""
                 CREATE TABLE IF NOT EXISTS server_lotto_state (
                                                                   server_id INTEGER PRIMARY KEY,
                                                                   winning_number INTEGER,
                                                                   jackpot INTEGER,
                                                                   last_bonus_date TEXT DEFAULT ''
                 );""")

    c.execute("""CREATE TABLE IF NOT EXISTS user_housing (
                                                             user_id INTEGER PRIMARY KEY,
                                                             house_type TEXT,
                                                             level INTEGER DEFAULT 1,
                                                             last_collected TIMESTAMP,
                                                             custom_name TEXT DEFAULT NULL,
                                                             custom_color TEXT DEFAULT NULL,
                                                             under_construction TEXT DEFAULT NULL,
                                                             finish_time TIMESTAMP DEFAULT NULL,
                                                             stored_items TEXT DEFAULT '{}',
                                                             FOREIGN KEY (user_id) REFERENCES users(user_id)
                 );""")

    # Migration for existing user_housing tables
    new_housing_cols = [
        ("custom_name", "TEXT DEFAULT NULL"),
        ("custom_color", "TEXT DEFAULT NULL"),
        ("under_construction", "TEXT DEFAULT NULL"),
        ("finish_time", "TIMESTAMP DEFAULT NULL"),
        ("stored_items", "TEXT DEFAULT '{}'")
    ]
    for col, type_def in new_housing_cols:
        try:
            conn.execute(f"ALTER TABLE user_housing ADD COLUMN {col} {type_def}")
        except sqlite3.OperationalError:
            pass

    c.execute("""
              CREATE TABLE IF NOT EXISTS user_housing_upgrades (
                                                                   user_id INTEGER,
                                                                   upgrade_id TEXT,
                                                                   PRIMARY KEY (user_id, upgrade_id),
                                                                   FOREIGN KEY (user_id) REFERENCES users(user_id)
              );""")

    c.execute("""
              CREATE TABLE IF NOT EXISTS pet_expeditions (
                                                             id INTEGER PRIMARY KEY AUTOINCREMENT,
                                                             user_id INTEGER,
                                                             pet_id INTEGER,
                                                             start_time TIMESTAMP,
                                                             end_time TIMESTAMP,
                                                             reward_xp INTEGER DEFAULT 0,
                                                             reward_items TEXT, -- JSON list of item names
                                                             log TEXT,          -- JSON list of event objects
                                                             is_claimed BOOLEAN DEFAULT 0,
                                                             FOREIGN KEY (pet_id) REFERENCES user_pets(id)
              );""")

    # Migration for existing user_pets
    try:
        conn.execute("ALTER TABLE user_pets ADD COLUMN on_expedition BOOLEAN DEFAULT 0")
    except sqlite3.OperationalError:
        pass

    c.execute("""CREATE TABLE IF NOT EXISTS user_farming
                 (
                     id INTEGER PRIMARY KEY AUTOINCREMENT,
                     user_id INTEGER,
                     zone_key TEXT,
                     plot_index INTEGER,
                     item_name TEXT,
                     plant_time TIMESTAMP,
                     grow_time INTEGER,
                     FOREIGN KEY (user_id) REFERENCES users(user_id)
                 );""")

    # --- Quest System Tables ---
    c.execute("""CREATE TABLE IF NOT EXISTS user_quests (
                                                            user_id INTEGER,
                                                            quest_id TEXT,
                                                            status TEXT DEFAULT 'ACTIVE', -- 'ACTIVE', 'COMPLETED', 'FAILED'
                                                            started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                                            completed_at TIMESTAMP,
                                                            PRIMARY KEY (user_id, quest_id),
                                                            FOREIGN KEY (user_id) REFERENCES users(user_id)
                 );""")

    c.execute("""CREATE TABLE IF NOT EXISTS user_quest_data (
                                                                user_id INTEGER,
                                                                quest_id TEXT,
                                                                step_index INTEGER DEFAULT 0,
                                                                progress_value INTEGER DEFAULT 0,
                                                                custom_data TEXT DEFAULT '{}', -- JSON blob for choices, etc.
                                                                PRIMARY KEY (user_id, quest_id),
                                                                FOREIGN KEY (user_id, quest_id) REFERENCES user_quests(user_id, quest_id) ON DELETE CASCADE
                 );""")

    # --- NPC System Tables ---
    c.execute("""CREATE TABLE IF NOT EXISTS user_npc_reputation (
                                                                    user_id INTEGER,
                                                                    npc_id TEXT,
                                                                    reputation INTEGER DEFAULT 0,
                                                                    level INTEGER DEFAULT 1,
                                                                    PRIMARY KEY (user_id, npc_id),
                                                                    FOREIGN KEY (user_id) REFERENCES users(user_id)
                 );""")

    c.execute("""CREATE TABLE IF NOT EXISTS user_npc_daily_rep (
                                                                   user_id INTEGER,
                                                                   npc_id TEXT,
                                                                   date_str TEXT,
                                                                   amount INTEGER DEFAULT 0,
                                                                   PRIMARY KEY (user_id, npc_id, date_str),
                                                                   FOREIGN KEY (user_id) REFERENCES users(user_id)
                 );""")

    # --- Community System Tables ---
    c.execute("""CREATE TABLE IF NOT EXISTS server_projects (
                                                                server_id INTEGER,
                                                                project_id TEXT,
                                                                level INTEGER DEFAULT 1,
                                                                PRIMARY KEY (server_id, project_id)
                 );""")

    c.execute("""CREATE TABLE IF NOT EXISTS server_project_contributions (
                                                                             server_id INTEGER,
                                                                             project_id TEXT,
                                                                             resource_type TEXT,
                                                                             amount_contributed INTEGER DEFAULT 0,
                                                                             PRIMARY KEY (server_id, project_id, resource_type)
                 );""")

    c.execute("""CREATE TABLE IF NOT EXISTS user_community_stats (
                                                                     user_id INTEGER,
                                                                     server_id INTEGER,
                                                                     total_money_invested INTEGER DEFAULT 0,
                                                                     total_items_invested INTEGER DEFAULT 0,
                                                                     PRIMARY KEY (user_id, server_id),
                                                                     FOREIGN KEY (user_id) REFERENCES users(user_id)
                 );""")

    conn.commit()
    conn.close()
    print("Base de données initialisée avec succès.")
