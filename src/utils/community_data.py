class CommunityBuilding:
    def __init__(self, key: str, max_level: int, get_cost_func, get_bonuses_func):
        self.key = key
        self.max_level = max_level
        self.get_cost = get_cost_func
        self.get_bonuses = get_bonuses_func

def market_cost(level: int) -> dict:
    base = {"money": 10000, "Pebble": 200}
    if level == 2:
        return {"money": 50000, "Pebble": 1000, "Wood": 200}
    elif level == 3:
        return {"money": 150000, "Wood": 500, "Stone": 500}
    elif level == 4:
        return {"money": 500000, "Stone": 1500, "Iron Ore": 500}
    elif level == 5:
        return {"money": 1000000, "Iron Ore": 2000, "Gold Ore": 200}
    elif level == 6:
        return {"money": 3000000, "Gold Ore": 1000, "Diamond": 50}
    elif level == 7:
        return {"money": 5000000, "Diamond": 200, "Ruby": 200}
    elif level == 8:
        return {"money": 8000000, "Ruby": 500, "Emerald": 500}
    elif level == 9:
        return {"money": 15000000, "Emerald": 1000, "Ancient Relic": 50}
    elif level == 10:
        return {"money": 30000000, "Ancient Relic": 200, "Meteorite": 10}
    return base

def market_bonuses(level: int) -> dict:
    if level == 0:
        return {}
    return {"shop_discount": min(20, level * 2)} # max 20%

def bank_cost(level: int) -> dict:
    if level == 1:
        return {"money": 15000, "Coal": 100}
    elif level == 2:
        return {"money": 60000, "Iron Ore": 100}
    elif level == 3:
        return {"money": 200000, "Gold Ore": 100}
    elif level == 4:
        return {"money": 700000, "Gold Ore": 500}
    elif level == 5:
        return {"money": 2000000, "Diamond": 100}
    elif level == 6:
        return {"money": 5000000, "Ruby": 100}
    elif level == 7:
        return {"money": 10000000, "Emerald": 100}
    elif level == 8:
        return {"money": 20000000, "Sapphire": 100}
    elif level == 9:
        return {"money": 50000000, "Star Fragment": 10}
    elif level == 10:
        return {"money": 100000000, "Star Fragment": 50}
    return {"money": 15000, "Coal": 100}

def bank_bonuses(level: int) -> dict:
    if level == 0:
        return {}
    return {"job_payout": min(50, level * 5)} # max 50%

def statue_cost(level: int) -> dict:
    base_money = 10000 * (5 ** (level - 1))
    base_pebble = 500 * (level)
    return {"money": base_money, "Pebble": base_pebble}

def statue_bonuses(level: int) -> dict:
    if level == 0:
        return {}
    return {"glory_bonus": level * 10} 

BUILDINGS = {
    "market": CommunityBuilding("market", 10, market_cost, market_bonuses),
    "bank": CommunityBuilding("bank", 10, bank_cost, bank_bonuses),
    "statue": CommunityBuilding("statue", 5, statue_cost, statue_bonuses)
}

def get_building(key: str) -> CommunityBuilding:
    return BUILDINGS.get(key)
