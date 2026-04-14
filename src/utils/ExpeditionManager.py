import random

from src.models.Pet import Pet, PETS_DB, PetBonus
from src.utils.i18n import t, get_pet_name, get_item_name

# Possible locations for exploration events
LOCATIONS = [
    "forest", "desert", "cave", 
    "plains", "mountain", "swamp",
    "valley", "coral", "volcano"
]

# Possible random events
EVENT_TYPES = ["exploration", "combat", "loot", "rest"]

# Possible common items for loot events
COMMON_LOOT = ["caillou", "charbon", "sardine", "blé", "tomate"]
RARE_LOOT = ["minerai de fer", "saumon", "maïs", "fraise"]
EPIC_LOOT = ["pépite d'or", "requin", "fruit étoile", "emeraude"]

def generate_expedition(pet: Pet, duration_hours: int, lang: str = 'fr'):
    events = []
    total_xp = 0
    looted_items = []
    
    num_events = duration_hours * 2
    if duration_hours == 1: num_events = 3
    
    total_xp += duration_hours * 25
    
    for i in range(num_events):
        event_time_mins = int((i + 1) * (duration_hours * 60) / (num_events + 1))
        
        event_type = random.choices(EVENT_TYPES, weights=[40, 30, 20, 10])[0]
        
        event_data = {"time": event_time_mins, "type": event_type}
        
        if event_type == "exploration":
            loc_key = random.choice(LOCATIONS)
            loc_name = t(f"expedition.locations.{loc_key}", lang)
            xp = random.randint(10, 30) * pet.level
            total_xp += xp
            event_data["text"] = t("expedition.events.exploration", lang, pet=pet.nickname, location=loc_name, xp=xp)
            event_data["xp"] = xp
            
        elif event_type == "combat":
            enemy_species = random.choice(list(PETS_DB.keys()))
            enemy_lvl = max(1, pet.level + random.randint(-2, 2))
            
            # Simple win/loss simulation
            win_chance = 0.6 + (pet.level - enemy_lvl) * 0.05
            if pet.bonus == PetBonus.HUNT:
                win_chance += 0.1
            win_chance = max(0.2, min(0.95, win_chance))
            
            if random.random() < win_chance:
                xp = random.randint(40, 80) * enemy_lvl
                total_xp += xp
                event_data["text"] = t("expedition.events.combat_win", lang, pet=pet.nickname, enemy=get_pet_name(enemy_species, lang), xp=xp)
                event_data["xp"] = xp
                # Small chance for loot on win
                if random.random() < 0.3:
                    item = random.choice(COMMON_LOOT if random.random() < 0.8 else RARE_LOOT)
                    looted_items.append(item)
                    event_data["loot"] = item
            else:
                event_data["text"] = t("expedition.events.combat_loss", lang, pet=pet.nickname, enemy=get_pet_name(enemy_species, lang))
                total_xp += 10
                
        elif event_type == "loot":
            roll = random.random()
            if roll < 0.05: item = random.choice(EPIC_LOOT)
            elif roll < 0.25: item = random.choice(RARE_LOOT)
            else: item = random.choice(COMMON_LOOT)
            
            looted_items.append(item)
            event_data["text"] = t("expedition.events.loot", lang, pet=pet.nickname, item=get_item_name(item, lang))
            event_data["loot"] = item
            
        elif event_type == "rest":
            event_data["text"] = t("expedition.events.rest", lang, pet=pet.nickname)
            
        events.append(event_data)
        
    return {
        "log": events,
        "xp": total_xp,
        "items": looted_items
    }
