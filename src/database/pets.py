from src.database.db_handler import get_connection
from src.models.Pet import Pet


def insert_new_pet(pet: Pet):
    conn = get_connection()
    try:
        cursor = conn.execute("""
                              INSERT INTO user_pets
                              (user_id, pet_type, nickname, level, xp, max_hp, hp, atk, defense, speed, dge, acc,
                               crit_c, crit_d, elo, bonus, is_active)
                              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                              """, (
                                  pet.user_id, pet.pet_type, pet.nickname, pet.level, pet.xp, pet.max_hp, pet.hp,
                                  pet.atk, pet.defense, pet.speed, pet.dge, pet.acc, pet.crit_c, pet.crit_d,
                                  pet.elo, pet.bonus_type, pet.is_active
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
                         food_eaten=?
                     WHERE id = ?
                     """, (
                         pet.level, pet.nickname, pet.xp, pet.max_hp, pet.hp, pet.atk, pet.defense, pet.speed,
                         pet.dge, pet.acc, pet.crit_c, pet.crit_d, pet.spc_c, pet.trs_lvl, pet.elo, pet.bonus_type, pet.food_eaten,
                         pet.id
                     ))
        conn.commit()
    finally:
        conn.close()


def get_active_pet(user_id) -> Pet:
    conn = get_connection()
    try:
        row = conn.execute("SELECT * FROM user_pets WHERE user_id = ? AND is_active = 1", (user_id,)).fetchone()
        if row:
            return Pet.from_db(dict(row))
        return None
    finally:
        conn.close()


def get_all_pets(user_id) -> list[Pet]:
    conn = get_connection()
    try:
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


def get_pet_by_id(pet_id: int) -> Pet:
    conn = get_connection()
    try:
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


def get_random_pets(limit: int = 2, min_lvl=1) -> list[Pet]:
    conn = get_connection()
    try:
        rows = conn.execute("SELECT * FROM user_pets WHERE level >= ? ORDER BY RANDOM() LIMIT ?", (min_lvl, limit)).fetchall()
        return [Pet.from_db(dict(row)) for row in rows]
    finally:
        conn.close()


def get_random_pet_and_opponent(min_lvl=1, elo_range=50) -> list[Pet]:
    conn = get_connection()
    try:
        pet1_row = conn.execute("SELECT * FROM user_pets WHERE level >= ? ORDER BY RANDOM() LIMIT 1", (min_lvl,)).fetchone()
        if not pet1_row:
            return []
            
        pet2_row = conn.execute(
            "SELECT * FROM user_pets WHERE level >= ? AND id != ? AND ABS(elo - ?) <= ? ORDER BY RANDOM() LIMIT 1", 
            (min_lvl, pet1_row['id'], pet1_row['elo'], elo_range)
        ).fetchone()
        
        if not pet2_row:
            pet2_row = conn.execute(
                "SELECT * FROM user_pets WHERE level >= ? AND id != ? ORDER BY ABS(elo - ?) ASC, RANDOM() LIMIT 1", 
                (min_lvl, pet1_row['id'], pet1_row['elo'])
            ).fetchone()
            
        if not pet2_row:
            return [Pet.from_db(dict(pet1_row))]
            
        return [Pet.from_db(dict(pet1_row)), Pet.from_db(dict(pet2_row))]
    finally:
        conn.close()


def update_pet_elo(pet_id: int, elo: int):
    conn = get_connection()
    try:
        conn.execute("UPDATE user_pets SET elo = ? WHERE id = ?", (elo, pet_id))
        conn.commit()
    finally:
        conn.close()


def get_pet_rank(pet_id: int) -> dict:
    conn = get_connection()
    try:
        rows = conn.execute("SELECT id, elo FROM user_pets WHERE level >= 5 ORDER BY elo DESC, id ASC").fetchall()
    finally:
        conn.close()

    if not rows:
        return {"rank": "Non classé", "progress": 0}

    all_pets = [dict(row) for row in rows]
    
    pet_index = next((i for i, p in enumerate(all_pets) if p['id'] == pet_id), -1)
    if pet_index == -1:
        return {"rank": "Non classé", "progress": 0}
        
    pet_elo = all_pets[pet_index]['elo']
    
    if len(all_pets) <= 5 or pet_index < 5:
        return {"rank": "Top 5 🌟", "progress": 100}
        
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
    
    return {"rank": rank_name, "progress": int(progress)}
