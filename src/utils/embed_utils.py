def generate_hp_bar(current_hp, max_hp, length=10):
    if max_hp <= 0:
        max_hp = 1
    percent = max(0.0, min(1.0, current_hp / max_hp))
    filled = int(length * percent)
    empty = length - filled
    return "🟩" * filled + "🟥" * empty

