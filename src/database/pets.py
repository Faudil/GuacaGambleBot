from src.database.db_handler import get_connection
from src.models.Pet import Pet


def insert_new_pet(pet: Pet):
    from src.database.housing import can_add_pet
    can, current, limit = can_add_pet(pet.user_id)
    if not can:
        return None
        
    conn = get_connection()
    try:
        cursor = conn.execute("""
                              INSERT INTO user_pets
                              (user_id, pet_type, nickname, level, xp, max_hp, hp, atk, defense, speed, dge, acc,
                                crit_c, crit_d, elo, bonus, is_active, on_expedition)
                              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                              """, (
                                  pet.user_id, pet.pet_type, pet.nickname, pet.level, pet.xp, pet.max_hp, pet.hp,
                                  pet.atk, pet.defense, pet.speed, pet.dge, pet.acc, pet.crit_c, pet.crit_d,
                                  pet.elo, pet.bonus_type, pet.is_active, pet.on_expedition
                              ))
        conn.commit()
        pet.id = cursor.lastrowid
        return pet
    finally:
        conn.close()


def update_pet(pet: Pet):
    conn = get_connection()
    try:
        conn.execute("""
                     UPDATE user_pets
                     SET level=?,
                         nickname=?,
                         xp=?,
                         max_hp=?,
                         hp=?,
                         atk=?,
                         defense=?,
                         speed=?,
                         dge=?,
                         acc=?,
                         crit_c=?,
                         crit_d=?,
                         spc_c=?,
                         trs_lvl=?,
                         elo=?,
                         bonus=?,
                         food_eaten=?,
                         on_expedition=?
                     WHERE id = ?
                     """, (
                         pet.level, pet.nickname, pet.xp, pet.max_hp, pet.hp, pet.atk, pet.defense, pet.speed,
                         pet.dge, pet.acc, pet.crit_c, pet.crit_d, pet.spc_c, pet.trs_lvl, pet.elo, pet.bonus_type, pet.food_eaten,
                         pet.on_expedition, pet.id
                     ))
        conn.commit()
    finally:
        conn.close()


def get_active_pet(user_id, server_id=None) -> Pet:
    conn = get_connection()
    try:
        if server_id:
            row = conn.execute("""
                SELECT up.*, COALESCE(spe.elo, up.elo) as elo
                FROM user_pets up
                LEFT JOIN server_pet_elo spe ON up.id = spe.pet_id AND spe.server_id = ?
                WHERE up.user_id = ? AND up.is_active = 1
                """, (server_id, user_id)).fetchone()
        else:
            row = conn.execute("SELECT * FROM user_pets WHERE user_id = ? AND is_active = 1", (user_id,)).fetchone()
        if row:
            return Pet.from_db(dict(row))
        return None
    finally:
        conn.close()


def get_all_pets(user_id, server_id=None) -> list[Pet]:
    conn = get_connection()
    try:
        if server_id:
            rows = conn.execute("""
                SELECT up.*, COALESCE(spe.elo, up.elo) as elo
                FROM user_pets up
                LEFT JOIN server_pet_elo spe ON up.id = spe.pet_id AND spe.server_id = ?
                WHERE up.user_id = ?
                """, (server_id, user_id)).fetchall()
        else:
            rows = conn.execute("SELECT * FROM user_pets WHERE user_id = ?", (user_id,)).fetchall()
        return [Pet.from_db(dict(row)) for row in rows]
    finally:
        conn.close()


def set_active_pet(user_id, pet_id):
    conn = get_connection()
    try:
        conn.execute("UPDATE user_pets SET is_active = 0 WHERE user_id = ?", (user_id,))
        cursor = conn.execute("UPDATE user_pets SET is_active = 1 WHERE id = ? AND user_id = ?", (pet_id, user_id))
        conn.commit()
        return cursor.rowcount > 0
    finally:
        conn.close()


def get_pet_by_id(pet_id: int, server_id=None) -> Pet:
    conn = get_connection()
    try:
        if server_id:
            row = conn.execute("""
                SELECT up.*, COALESCE(spe.elo, up.elo) as elo
                FROM user_pets up
                LEFT JOIN server_pet_elo spe ON up.id = spe.pet_id AND spe.server_id = ?
                WHERE up.id = ?
                """, (server_id, pet_id)).fetchone()
        else:
            row = conn.execute("SELECT * FROM user_pets WHERE id = ?", (pet_id,)).fetchone()
        if row:
            return Pet.from_db(dict(row))
        return None
    finally:
        conn.close()


def transfer_pet(pet_id: int, new_owner_id: int):
    conn = get_connection()
    try:
        conn.execute("UPDATE user_pets SET user_id = ?, is_active = 0 WHERE id = ?", (new_owner_id, pet_id))
        conn.commit()
    finally:
        conn.close()


def delete_pet(pet_id: int):
    conn = get_connection()
    try:
        conn.execute("DELETE FROM user_pets WHERE id = ?", (pet_id,))
        conn.commit()
    finally:
        conn.close()


def get_random_pets(limit: int = 2, min_lvl=1) -> list[Pet]:
    conn = get_connection()
    try:
        rows = conn.execute("SELECT * FROM user_pets WHERE level >= ? ORDER BY RANDOM() LIMIT ?", (min_lvl, limit)).fetchall()
        return [Pet.from_db(dict(row)) for row in rows]
    finally:
        conn.close()


def get_random_pet_and_opponent(server_id: int, min_lvl=1, elo_range=50) -> list[Pet]:
    conn = get_connection()
    try:
        pet1_row = conn.execute("""
            SELECT up.*, COALESCE(spe.elo, up.elo) as elo
            FROM user_pets up
            LEFT JOIN server_pet_elo spe ON up.id = spe.pet_id AND spe.server_id = ?
            WHERE up.level >= ? AND up.is_active = 1
            ORDER BY RANDOM() LIMIT 1
            """, (server_id, min_lvl)).fetchone()
        
        if not pet1_row:
            return []
            
        pet2_row = conn.execute("""
            SELECT up.*, COALESCE(spe.elo, up.elo) as elo
            FROM user_pets up
            LEFT JOIN server_pet_elo spe ON up.id = spe.pet_id AND spe.server_id = ?
            WHERE up.level >= ? AND up.id != ? AND ABS(COALESCE(spe.elo, up.elo) - ?) <= ? AND up.is_active = 1
            ORDER BY RANDOM() LIMIT 1
            """, (server_id, min_lvl, pet1_row['id'], pet1_row['elo'], elo_range)).fetchone()
        
        if not pet2_row:
            pet2_row = conn.execute("""
                SELECT up.*, COALESCE(spe.elo, up.elo) as elo
                FROM user_pets up
                LEFT JOIN server_pet_elo spe ON up.id = spe.pet_id AND spe.server_id = ?
                WHERE up.level >= ? AND up.id != ? AND up.is_active = 1
                ORDER BY ABS(COALESCE(spe.elo, up.elo) - ?) ASC, RANDOM() LIMIT 1
                """, (server_id, min_lvl, pet1_row['id'], pet1_row['elo'])).fetchone()
            
        if not pet2_row:
            return [Pet.from_db(dict(pet1_row))]
            
        return [Pet.from_db(dict(pet1_row)), Pet.from_db(dict(pet2_row))]
    finally:
        conn.close()


def update_pet_elo(pet_id: int, server_id: int, elo: int):
    conn = get_connection()
    try:
        conn.execute("""
            INSERT INTO server_pet_elo (pet_id, server_id, elo)
            VALUES (?, ?, ?)
            ON CONFLICT(pet_id, server_id) DO UPDATE SET elo = excluded.elo
            """, (pet_id, server_id, elo))
        conn.commit()
    finally:
        conn.close()


RANK_GLORY = {
    "Top 5 🌟": 500,
    "Diamant 💎": 300,
    "Or 🥇": 150,
    "Argent 🥈": 50,
    "Bronze 🥉": 10
}


def get_all_pet_ranks(server_id: int = None) -> dict:
    conn = get_connection()
    try:
        if server_id:
            rows = conn.execute("""
                SELECT up.id, up.user_id, COALESCE(spe.elo, up.elo) as elo
                FROM user_pets up
                LEFT JOIN server_pet_elo spe ON up.id = spe.pet_id AND spe.server_id = ?
                WHERE up.level >= 5
                ORDER BY COALESCE(spe.elo, up.elo) DESC, up.id ASC
            """, (server_id,)).fetchall()
        else:
            rows = conn.execute("SELECT id, user_id, elo FROM user_pets WHERE level >= 5 ORDER BY elo DESC, id ASC").fetchall()
    finally:
        conn.close()

    if not rows:
        return {}

    all_pets = [dict(row) for row in rows]
    ranks = {}
    
    max_elo_all = all_pets[0]['elo']
    min_elo_all = all_pets[-1]['elo']
    
    for pet_index, pet in enumerate(all_pets):
        pet_elo = pet['elo']
        pet_id = pet['id']
        
        if max_elo_all - min_elo_all < 600 or len(all_pets) < 10:
            relative_elo = pet_elo - min_elo_all
            rank_group = min(3, (relative_elo // 200 + 1))
            rank_name = {3: "Or 🥇", 2: "Argent 🥈", 1: "Bronze 🥉"}[rank_group]
            progress = (relative_elo % 200) // 2
        else:
            if pet_index < 5:
                rank_name = "Top 5 🌟"
                progress = 100.0
            else:
                rest_pets = all_pets[5:]
                N = len(rest_pets)
                pet_rest_index = pet_index - 5
                pet_group = (pet_rest_index * 4) // N
                
                group_elos = [p['elo'] for i, p in enumerate(rest_pets) if (i * 4) // N == pet_group]
                min_elo = min(group_elos)
                max_elo = max(group_elos)
                
                if max_elo == min_elo:
                    progress = 100.0
                else:
                    progress = (pet_elo - min_elo) / (max_elo - min_elo) * 100.0
                    
                rank_name = {
                    0: "Diamant 💎",
                    1: "Or 🥇",
                    2: "Argent 🥈",
                    3: "Bronze 🥉"
                }[pet_group]
                
        ranks[pet_id] = {"rank": rank_name, "progress": int(progress), "user_id": pet['user_id']}
        
    return ranks


def get_pet_rank(pet_id: int, server_id: int = None) -> dict:
    ranks = get_all_pet_ranks(server_id)
    if pet_id in ranks:
        return {"rank": ranks[pet_id]["rank"], "progress": ranks[pet_id]["progress"]}
    return {"rank": "Non classé", "progress": 0}
