from src.database.db_handler import get_connection


def get_job_data(user_id, job_name):
    conn = get_connection()
    row = conn.execute("SELECT level, xp FROM jobs WHERE user_id = ? AND job_name = ?", (user_id, job_name)).fetchone()
    conn.close()
    if row:
        return row['level'], row['xp']
    return 1, 0


def add_job_xp(user_id, job_name, amount):
    current_lvl, current_xp = get_job_data(user_id, job_name)
    new_xp = current_xp + amount
    xp_needed = current_lvl * 100
    leveled_up = False
    new_lvl = current_lvl
    if new_xp >= xp_needed:
        new_xp -= xp_needed
        new_lvl += 1
        leveled_up = True

    conn = get_connection()
    conn.execute("""
                 INSERT INTO jobs (user_id, job_name, level, xp)
                 VALUES (?, ?, ?, ?)
                 ON CONFLICT(user_id, job_name) DO UPDATE SET level = ?,
                                                              xp    = ?
                 """, (user_id, job_name, new_lvl, new_xp, new_lvl, new_xp))
    conn.commit()
    conn.close()

    return leveled_up, new_lvl