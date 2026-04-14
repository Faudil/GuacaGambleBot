from src.database.db_handler import get_connection
import datetime

def get_user_plots(user_id, zone_key):
    conn = get_connection()
    c = conn.cursor()
    c.execute("SELECT * FROM user_farming WHERE user_id = ? AND zone_key = ?", (user_id, zone_key))
    plots = c.fetchall()
    conn.close()
    return plots

def plant_seed(user_id, zone_key, plot_index, item_name, grow_time):
    conn = get_connection()
    now = datetime.datetime.now()
    try:
        conn.execute("""
            INSERT INTO user_farming (user_id, zone_key, plot_index, item_name, plant_time, grow_time)
            VALUES (?, ?, ?, ?, ?, ?)
        """, (user_id, zone_key, plot_index, item_name, now, grow_time))
        conn.commit()
    except Exception as e:
        print(f"Error planting seed: {e}")
    finally:
        conn.close()

def harvest_plot(user_id, zone_key, plot_index):
    conn = get_connection()
    try:
        conn.execute("DELETE FROM user_farming WHERE user_id = ? AND zone_key = ? AND plot_index = ?", 
                     (user_id, zone_key, plot_index))
        conn.commit()
        return True
    except Exception as e:
        print(f"Error harvesting: {e}")
        return False
    finally:
        conn.close()

def get_all_user_plots(user_id):
    conn = get_connection()
    c = conn.cursor()
    c.execute("SELECT * FROM user_farming WHERE user_id = ?", (user_id,))
    plots = c.fetchall()
    conn.close()
    return plots
