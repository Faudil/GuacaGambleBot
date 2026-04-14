import random
import enum
from typing import Optional
from src.items.Item import ItemRarity
from src.utils.i18n import t, get_pet_name


class PetBonus(enum.Enum):
    HUNT = 0
    FARM = 1
    MINE = 2
    FISH = 3


class DamageType(enum.Enum):
    IMPACT = "impact"
    BITE = "morsure"
    SCRATCH = "griffure"
    POISON = "poison"
    FIRE = "feu"


class PetStatus(enum.Enum):
    STUNNED = 0
    BLEEDING = 1
    FIRE = 2
    POISONED = 3
    WEAKENED = 4
    FROZEN = 5


PET_DAMAGE_TYPES = {
    # Common
    "Escargot": DamageType.IMPACT,
    "Souris": DamageType.BITE,
    "Cochon": DamageType.IMPACT,
    "Grenouille": DamageType.IMPACT,
    "Taupe": DamageType.SCRATCH,
    "Pélican": DamageType.BITE,
    "Mouton": DamageType.IMPACT,
    "Abeille": DamageType.POISON,

    # Rare
    "Chien": DamageType.BITE,
    "Chat": DamageType.SCRATCH,
    "Cheval": DamageType.IMPACT,
    "Renard": DamageType.BITE,
    "Singe": DamageType.BITE,
    "Ours": DamageType.BITE,

    # Epic
    "Chameau": DamageType.IMPACT,
    "Panda": DamageType.BITE,
    "Tigre": DamageType.SCRATCH,
    "Pieuvre": DamageType.POISON,

    # Legendary
    "Dragon": DamageType.FIRE,

# Common
    "Hamster": DamageType.FIRE,
    "Fourmi": DamageType.BITE,
    "Hérisson": DamageType.SCRATCH,
    "Canard": DamageType.BITE,
    "Chouette": DamageType.SCRATCH,
    "Paresseux": DamageType.SCRATCH,


# Rare
    "Kangourou": DamageType.IMPACT,
    "Iguane": DamageType.POISON,
    "Gorille": DamageType.IMPACT,
    "Scorpion": DamageType.POISON,
    "Bison": DamageType.IMPACT,


# Epic
    "Rhino": DamageType.IMPACT,
    "Aigle": DamageType.SCRATCH,
    "Crocodile": DamageType.BITE,
    "Putois": DamageType.POISON,
    "Dauphin": DamageType.IMPACT,
    "Léopard": DamageType.BITE,
    "Lion": DamageType.SCRATCH,
    "Ours polaire": DamageType.BITE,


# Legendary
    "Tyrannosaure": DamageType.BITE,
    "Diplodocus": DamageType.IMPACT,
    "Mamouth": DamageType.IMPACT,
    "Mégalodon": DamageType.BITE,
    "Kraken": DamageType.POISON,
    "Licorne": DamageType.FIRE,
    "Phoenix": DamageType.FIRE,
    "Cerbère": DamageType.FIRE,
    "Fenrir": DamageType.SCRATCH,
    "Ratatosk": DamageType.SCRATCH,
    "Nidhögg": DamageType.POISON,
    "Bedawang": DamageType.POISON,
}




PETS_DB = {
    "Escargot": {
        "rarity": ItemRarity.common, "emoji": "🐌", "bonus": PetBonus.FARM,
        "hp": 60, "attack": 5, "defense": 15, "speed": 2,
        "dge": 0, "acc": 10, "crit_chance": 5, "crit_dmg": 1.2
    },

    "Souris": {
        "rarity": ItemRarity.common, "emoji": "🐀", "bonus": PetBonus.MINE,
        "hp": 25, "attack": 12, "defense": 3, "speed": 25,
        "dge": 25, "acc": 5, "crit_chance": 10, "crit_dmg": 1.5
    },

    "Cochon": {
        "rarity": ItemRarity.common, "emoji": "🐷", "bonus": PetBonus.FARM,
        "hp": 60, "attack": 12, "defense": 8, "speed": 8,
        "dge": 2, "acc": 10, "crit_chance": 5, "crit_dmg": 1.5
    },

    "Grenouille": {
        "rarity": ItemRarity.common, "emoji": "🐸", "bonus": PetBonus.FISH,
        "hp": 30, "attack": 15, "defense": 4, "speed": 20,
        "dge": 10, "acc": 10, "crit_chance": 8, "crit_dmg": 1.5
    },

    "Taupe": {
        "rarity": ItemRarity.common, "emoji": "🦡", "bonus": PetBonus.MINE,
        "hp": 45, "attack": 14, "defense": 8, "speed": 8,
        "dge": 5, "acc": 25, "crit_chance": 10, "crit_dmg": 2.0
    },

    "Pélican": {
        "rarity": ItemRarity.common, "emoji": "🦤", "bonus": PetBonus.FISH,
        "hp": 45, "attack": 12, "defense": 5, "speed": 18,
        "dge": 8, "acc": 15, "crit_chance": 5, "crit_dmg": 1.5
    },

    "Mouton": {
        "rarity": ItemRarity.common, "emoji": "🐑", "bonus": PetBonus.FARM,
        "hp": 55, "attack": 8, "defense": 12, "speed": 10,
        "dge": 5, "acc": 5, "crit_chance": 5, "crit_dmg": 1.5
    },
    "Abeille": {
        "rarity": ItemRarity.common, "emoji": "🐝", "bonus": PetBonus.FARM,
        "hp": 25, "attack": 24, "defense": 3, "speed": 25,
        "dge": 25, "acc": 5, "crit_chance": 10, "crit_dmg": 1.5
    },

    # RARE
    "Chien": {
        "rarity": ItemRarity.rare, "emoji": "🐶", "bonus": PetBonus.HUNT,
        "hp": 65, "attack": 22, "defense": 10, "speed": 18,
        "dge": 8, "acc": 15, "crit_chance": 10, "crit_dmg": 1.5
    },

    "Chat": {
        "rarity": ItemRarity.rare, "emoji": "😼", "bonus": PetBonus.FISH,
        "hp": 45, "attack": 25, "defense": 2, "speed": 35,
        "dge": 20, "acc": 10, "crit_chance": 20, "crit_dmg": 1.8
    },

    "Cheval": {
        "rarity": ItemRarity.rare, "emoji": "🐴", "bonus": PetBonus.FARM,
        "hp": 80, "attack": 18, "defense": 10, "speed": 30,
        "dge": 10, "acc": 10, "crit_chance": 5, "crit_dmg": 1.5
    },

    "Renard": {
        "rarity": ItemRarity.rare, "emoji": "🦊", "bonus": PetBonus.MINE,
        "hp": 50, "attack": 20, "defense": 6, "speed": 25,
        "dge": 18, "acc": 20, "crit_chance": 15, "crit_dmg": 1.6
    },

    "Singe": {
        "rarity": ItemRarity.rare, "emoji": "🐵", "bonus": PetBonus.FARM,
        "hp": 55, "attack": 22, "defense": 12, "speed": 28,
        "dge": 15, "acc": 15, "crit_chance": 12, "crit_dmg": 1.5
    },

    "Ours": {
        "rarity": ItemRarity.rare, "emoji": "🐻", "bonus": PetBonus.MINE,
        "hp": 90, "attack": 28, "defense": 18, "speed": 8,
        "dge": 2, "acc": 10, "crit_chance": 5, "crit_dmg": 2.0
    },

    # EPIC
    "Chameau": {
        "rarity": ItemRarity.epic, "emoji": "🐪", "bonus": PetBonus.FARM,
        "hp": 120, "attack": 18, "defense": 20, "speed": 12,
        "dge": 5, "acc": 10, "crit_chance": 5, "crit_dmg": 1.5
    },

    "Panda": {
        "rarity": ItemRarity.epic, "emoji": "🐼", "bonus": PetBonus.FARM,
        "hp": 110, "attack": 22, "defense": 15, "speed": 10,
        "dge": 8, "acc": 15, "crit_chance": 10, "crit_dmg": 1.5
    },

    "Tigre": {
        "rarity": ItemRarity.epic, "emoji": "🐯", "bonus": PetBonus.MINE,
        "hp": 85, "attack": 35, "defense": 12, "speed": 32,
        "dge": 15, "acc": 20, "crit_chance": 25, "crit_dmg": 2.0
    },

    "Pieuvre": {
        "rarity": ItemRarity.epic, "emoji": "🐙", "bonus": PetBonus.FISH,
        "hp": 100, "attack": 25, "defense": 15, "speed": 20,
        "dge": 25, "acc": 30, "crit_chance": 15, "crit_dmg": 1.5
    },

    # LEGENDARY
    "Dragon": {
        "rarity": ItemRarity.legendary, "emoji": "🐉", "bonus": PetBonus.HUNT,
        "hp": 130, "attack": 35, "defense": 20, "speed": 20,
        "dge": 15, "acc": 25, "crit_chance": 10, "crit_dmg": 1.2
    },

# COMMON AGAIN

    "Hamster": {
        "rarity": ItemRarity.common, "emoji": "🐹", "bonus": PetBonus.FARM,
        "hp": 25, "attack": 5, "defense": 10, "speed": 25,
        "dge": 22, "acc": 5, "crit_chance": 10, "crit_dmg": 1.5
    },

    "Fourmi": {
        "rarity": ItemRarity.common, "emoji": "🐜", "bonus": PetBonus.MINE,
        "hp": 5, "attack": 5, "defense": 5, "speed": 5,
        "dge": 5, "acc": 5, "crit_chance": 5, "crit_dmg": 1.2
    },

    "Hérisson": {
        "rarity": ItemRarity.common, "emoji": "🦔", "bonus": PetBonus.MINE,
        "hp": 28, "attack": 5, "defense": 26, "speed": 18,
        "dge": 5, "acc": 5, "crit_chance": 5, "crit_dmg": 1.5
    },

    "Canard": {
        "rarity": ItemRarity.common, "emoji": "🦆", "bonus": PetBonus.FISH,
        "hp": 30, "attack": 10, "defense": 8, "speed": 21,
        "dge": 9, "acc": 17, "crit_chance": 5, "crit_dmg": 1.7
    },

    "Chouette": {
        "rarity": ItemRarity.common, "emoji": "🦉", "bonus": PetBonus.HUNT,
        "hp": 35, "attack": 15, "defense": 5, "speed": 25,
        "dge": 8, "acc": 20, "crit_chance": 10, "crit_dmg": 1.5
    },

    "Paresseux": {
        "rarity": ItemRarity.common, "emoji": "🦥", "bonus": PetBonus.FISH,
        "hp": 50, "attack": 15, "defense": 15, "speed": 2,
        "dge": 0, "acc": 10, "crit_chance": 5, "crit_dmg": 1.2
    },

# RARE
    "Kangourou": {
        "rarity": ItemRarity.epic, "emoji": "🦘", "bonus": PetBonus.FARM,
        "hp": 65, "attack": 25, "defense": 15, "speed": 27,
        "dge": 18, "acc": 15, "crit_chance": 15, "crit_dmg": 2
    },

    "Iguane": {
        "rarity": ItemRarity.epic, "emoji": "🦎", "bonus": PetBonus.HUNT,
        "hp": 60, "attack": 20, "defense": 20, "speed": 20,
        "dge": 15, "acc": 18, "crit_chance": 18, "crit_dmg": 1.7
    },

    "Gorille": {
        "rarity": ItemRarity.rare, "emoji": "🦍", "bonus": PetBonus.FARM,
        "hp": 70, "attack": 22, "defense": 22, "speed": 12,
        "dge": 7, "acc": 12, "crit_chance": 10, "crit_dmg": 1.6
    },

    "Scorpion": {
        "rarity": ItemRarity.rare, "emoji": "🦂", "bonus": PetBonus.HUNT,
        "hp": 40, "attack": 22, "defense": 10, "speed": 25,
        "dge": 15, "acc": 15, "crit_chance": 25, "crit_dmg": 1.8
    },

    "Bison": {
        "rarity": ItemRarity.rare, "emoji": "🦬", "bonus": PetBonus.FARM,
        "hp": 80, "attack": 10, "defense": 18, "speed": 25,
        "dge": 8, "acc": 15, "crit_chance": 5, "crit_dmg": 1.6
    },

# EPIC

    "Aigle": {
        "rarity": ItemRarity.epic, "emoji": "🦅", "bonus": PetBonus.HUNT,
        "hp": 80, "attack": 30, "defense": 10, "speed": 35,
        "dge": 22, "acc": 25, "crit_chance": 20, "crit_dmg": 2.0
    },

    "Rhino": {
        "rarity": ItemRarity.epic, "emoji": "🦏", "bonus": PetBonus.MINE,
        "hp": 90, "attack": 26, "defense": 32, "speed": 12,
        "dge": 5, "acc": 10, "crit_chance": 12, "crit_dmg": 1.7
    },

    "Crocodile": {
        "rarity": ItemRarity.epic, "emoji": "🐊", "bonus": PetBonus.FISH,
        "hp": 80, "attack": 30, "defense": 20, "speed": 18,
        "dge": 17, "acc": 25, "crit_chance": 20, "crit_dmg": 2.0
    },

    "Putois": {
        "rarity": ItemRarity.common, "emoji": "🦨", "bonus": PetBonus.FARM,
        "hp": 90, "attack": 15, "defense": 26, "speed": 12,
        "dge": 24, "acc": 20, "crit_chance": 15, "crit_dmg": 1.7
    },

    "Dauphin": {
        "rarity": ItemRarity.epic, "emoji": "🐬", "bonus": PetBonus.FISH,
        "hp": 100, "attack": 18, "defense": 15, "speed": 32,
        "dge": 22, "acc": 30, "crit_chance": 20, "crit_dmg": 2.0
    },

    "Lépoard": {
        "rarity": ItemRarity.epic, "emoji": "🐆", "bonus": PetBonus.HUNT,
        "hp": 75, "attack": 35, "defense": 12, "speed": 46,
        "dge": 16, "acc": 18, "crit_chance": 20, "crit_dmg": 1.7
    },

    "Lion": {
        "rarity": ItemRarity.epic, "emoji": "🦁", "bonus": PetBonus.HUNT,
        "hp": 95, "attack": 35, "defense": 18, "speed": 18,
        "dge": 10, "acc": 22, "crit_chance": 20, "crit_dmg": 1.7
    },

    "Ours polaire": {
        "rarity": ItemRarity.rare, "emoji": "🐻‍❄️", "bonus": PetBonus.MINE,
        "hp": 100, "attack": 30, "defense": 20, "speed": 8,
        "dge": 2, "acc": 10, "crit_chance": 5, "crit_dmg": 2.0
    },


# Legendary

    "Tyrannosaure": {
        "rarity": ItemRarity.legendary, "emoji": "🦖", "bonus": PetBonus.HUNT,
        "hp": 120, "attack": 40, "defense": 20, "speed": 20,
        "dge": 15, "acc": 20, "crit_chance": 15, "crit_dmg": 2.0
    },

    "Diplodocus": {
        "rarity": ItemRarity.legendary, "emoji": "🦕", "bonus": PetBonus.FISH,
        "hp": 140, "attack": 20, "defense": 40, "speed": 15,
        "dge": 10, "acc": 20, "crit_chance": 10, "crit_dmg": 1.2
    },

    "Mamouth": {
        "rarity": ItemRarity.legendary, "emoji": "🦣", "bonus": PetBonus.MINE,
        "hp": 180, "attack": 20, "defense": 40, "speed": 10,
        "dge": 5, "acc": 10, "crit_chance": 5, "crit_dmg": 1.5
    },

    "Mégalodon": {
        "rarity": ItemRarity.legendary, "emoji": "🦈", "bonus": PetBonus.FISH,
        "hp": 130, "attack": 35, "defense": 25, "speed": 18,
        "dge": 10, "acc": 20, "crit_chance": 15, "crit_dmg": 1.5
    },

    "Kraken": {
        "rarity": ItemRarity.legendary, "emoji": "🦑", "bonus": PetBonus.FISH,
        "hp": 130, "attack": 25, "defense": 35, "speed": 18,
        "dge": 20, "acc": 10, "crit_chance": 15, "crit_dmg": 1.5
    },

    "Licorne": {
        "rarity": ItemRarity.legendary, "emoji": "🦄", "bonus": PetBonus.FARM,
        "hp": 100, "attack": 28, "defense": 20, "speed": 32,
        "dge": 20, "acc": 27, "crit_chance": 12, "crit_dmg": 1.2
    },

    "Phoenix": {
        "rarity": ItemRarity.legendary, "emoji": "🐦‍🔥", "bonus": PetBonus.FARM,
        "hp": 200, "attack": 20, "defense": 15, "speed": 30,
        "dge": 25, "acc": 15, "crit_chance": 15, "crit_dmg": 1.5
    },

    "Cerbère": {
        "rarity": ItemRarity.legendary, "emoji": "🐺🐺🐺", "bonus": PetBonus.HUNT,
        "hp": 90, "attack": 35, "defense": 20, "speed": 28,
        "dge": 15, "acc": 25, "crit_chance": 25, "crit_dmg": 1.5
    },

    "Fenrir": {
        "rarity": ItemRarity.legendary, "emoji": "🐺⛓️", "bonus": PetBonus.MINE,
        "hp": 100, "attack": 40, "defense": 20, "speed": 30,
        "dge": 20, "acc": 20, "crit_chance": 20, "crit_dmg": 1.5
    },

    "Ratatosk": {
        "rarity": ItemRarity.legendary, "emoji": "🐿️❄️", "bonus": PetBonus.MINE,
        "hp": 90, "attack": 30, "defense": 15, "speed": 40,
        "dge": 25, "acc": 30, "crit_chance": 25, "crit_dmg": 2.0
    },

    "Nidhögg": {
        "rarity": ItemRarity.legendary, "emoji": "🐍⚡", "bonus": PetBonus.MINE,
        "hp": 110, "attack": 32, "defense": 18, "speed": 20,
        "dge": 20, "acc": 50, "crit_chance": 18, "crit_dmg": 1.7
    },

    "Bedawang": {
        "rarity": ItemRarity.legendary, "emoji": "🐢🌳", "bonus": PetBonus.FARM,
        "hp": 200, "attack": 25, "defense": 40, "speed": 1,
        "dge": 0, "acc": 25, "crit_chance": 10, "crit_dmg": 1.2
    }
}

rarity_xp_multiplier = {
    ItemRarity.common: 0.5,
    ItemRarity.rare: 1,
    ItemRarity.epic: 1.5,
    ItemRarity.legendary: 2,
}

RARITY_FOOD_CAPACITY = {
    ItemRarity.common: 5,
    ItemRarity.rare: 4,
    ItemRarity.epic: 3,
    ItemRarity.legendary: 2
}


class Pet:
    def __init__(self, pet_id=None, user_id=None, pet_type="Escargot", nickname=None,
                 level=1, food_eaten=0, xp=0, max_hp=50, hp=50, atk=10, defense=5,
                 speed=10, dge=5, acc=0, crit_c=5, crit_d=1.5, elo=0,
                 bonus=0, spc_c=0, is_active=0, on_expedition=0, trs_lvl=0):

        self.id = pet_id
        self.user_id = user_id
        self.pet_type = pet_type
        self.nickname = nickname or pet_type

        self.level = level
        self.food_eaten = food_eaten
        self.xp = xp
        self.elo = elo
        self.is_active = is_active
        self.on_expedition = on_expedition
        self.bonus_type = bonus

        self.max_hp = max_hp
        self.hp = hp
        self.atk = atk
        self.defense = defense
        self.speed = speed
        self.dge = dge
        self.acc = acc
        self.crit_c = crit_c
        self.crit_d = crit_d
        self.spc_c = spc_c
        self.trs_lvl = trs_lvl

        self._defense_malus = 0
        self._acc_malus = 0
        self._dge_malus = 0
        self._atk_malus = 0
        self._speed_malus = 0
        self._stunned_turns = 0
        self._poisoned_turns = 0
        self._burning_turns = 0
        self._bleeding_turns = 0

        self._thorn_multiplier = 1



    @classmethod
    def from_db(cls, row: dict):
        return cls(
            pet_id=row.get('id'),
            user_id=row.get('user_id'),
            pet_type=row.get('pet_type'),
            nickname=row.get('nickname'),
            level=row.get('level', 1),
            food_eaten=row.get('food_eaten', 0),
            xp=row.get('xp', 0),
            max_hp=row.get('max_hp', 50),
            hp=row.get('hp', 50),
            atk=row.get('atk', 10),
            defense=row.get('defense', 5),
            speed=row.get('speed', 10),
            dge=row.get('dge', 5),
            acc=row.get('acc', 0),
            crit_c=row.get('crit_c', 5),
            crit_d=row.get('crit_d', 1.5),
            elo=row.get('elo', 1000),
            bonus=row.get('bonus', 0),
            spc_c=row.get('spc_c', 0),
            trs_lvl=row.get('trs_lvl', 0),
            is_active=row.get('is_active', 0),
            on_expedition=row.get('on_expedition', 0)
        )

    @classmethod
    def create_new(cls, user_id: int, pet_type: str, nickname: str = None):
        base = PETS_DB.get(pet_type)
        if not base:
            raise ValueError(f"L'espèce {pet_type} n'existe pas dans la base de données des familiers.")

        return cls(
            user_id=user_id,
            pet_type=pet_type,
            nickname=nickname or pet_type,
            max_hp=base['hp'],
            hp=base['hp'],
            atk=base['attack'],
            defense=base['defense'],
            speed=base['speed'],
            crit_c=base['crit_chance'],
            crit_d=base['crit_dmg'],
            bonus=base['bonus'].value,
            dge=base["dge"],
            acc=base["acc"],
        )

    @property
    def is_alive(self):
        return self.hp > 0

    @property
    def bonus(self):
        return PetBonus(self.bonus_type)

    @property
    def emoji(self):
        return PETS_DB.get(self.pet_type, {}).get("emoji", "🐾")

    @property
    def rarity(self) -> ItemRarity:
        return PETS_DB[self.pet_type]["rarity"]

    @property
    def max_food_capacity(self) -> int:
        return RARITY_FOOD_CAPACITY[self.rarity] * self.level

    def heal_full(self):
        self.hp = self.max_hp

    def display_name(self, lang: str = 'fr') -> str:
        return get_pet_name(self.pet_type, lang)

    def feed(self, item, lang: str = 'fr') -> Optional[str]:
        if self.food_eaten >= self.max_food_capacity:
            return t("pets.feed.full", lang, name=self.nickname, current=self.food_eaten, max=self.max_food_capacity)
        if not item.pet_effect:
            return t("pets.feed.refuse", lang, name=self.nickname, item=item.display_name(lang))
        stat_to_boost = item.pet_effect["stat"]
        boost_amount = item.pet_effect["amount"]
        current_stat_value = getattr(self, stat_to_boost, 0)
        new_value = current_stat_value + boost_amount
        if isinstance(new_value, float):
            new_value = round(new_value, 2)
        setattr(self, stat_to_boost, new_value)
        if stat_to_boost == "max_hp":
            self.hp += boost_amount
        self.food_eaten += 1
        return None

    def add_xp(self, amount: int):
        if self.level >= 20:
            self.xp = 0
            return False
        self.xp += amount
        leveled_up = False

        while self.xp >= self.level * rarity_xp_multiplier[self.rarity] * 100 and self.level < 20:
            self.xp -= self.level * rarity_xp_multiplier[self.rarity] * 100
            self.level += 1
            self.max_hp += 5
            self.hp += 5
            self.atk += 2
            self.defense += 1
            leveled_up = True
            if self.level == 5:
                self.elo = 1000
                self.spc_c = 5
            elif self.level == 10:
                self.spc_c = 10
            elif self.level == 20:
                self.spc_c = 15
        if self.level >= 20:
            self.xp = 0
        return leveled_up

    def forget_xp(self) -> bool:
        if self.level < 20:
            return False
        base = PETS_DB[self.pet_type]
        self.xp = 0
        self.level = 10
        self.max_hp = base["hp"] + 50
        self.hp = self.max_hp
        self.atk = base["attack"] + 20
        self.defense = base["defense"] + 10
        self.crit_c = base["crit_chance"]
        self.crit_d = base["crit_dmg"]
        self.acc = base["acc"]
        self.dge = base["dge"]
        self.speed = base["speed"]
        self.food_eaten = 0
        return True

    def tick_effects(self, lang: str = 'fr') -> str:
        msg_parts: list[str] = []
        if self._poisoned_turns > 0:
            dmg = max(1, int(self.max_hp * 0.05))
            self.hp = max(0, self.hp - dmg)
            msg_parts.append(t("pet_combat.poison_suffers", lang, pet=self.nickname, dmg=dmg))
            self._poisoned_turns -= 1
            if self._poisoned_turns == 0:
                self._atk_malus = 0
                msg_parts.append(t("pet_combat.poison_end", lang, pet=self.nickname))

        if self._burning_turns > 0:
            dmg = max(5, int(self.max_hp * 0.08))
            self.hp = max(0, self.hp - dmg)
            msg_parts.append(t("pet_combat.burn_suffers", lang, pet=self.nickname, dmg=dmg))
            self._burning_turns -= 1
            if self._burning_turns == 0:
                self._dge_malus = 0
                msg_parts.append(t("pet_combat.burn_end", lang, pet=self.nickname))

        if self._bleeding_turns > 0:
            dmg = max(2, int(self.max_hp * 0.06))
            self.hp = max(0, self.hp - dmg)
            msg_parts.append(t("pet_combat.bleed_suffers", lang, pet=self.nickname, dmg=dmg))
            self._bleeding_turns -= 1
            if self._bleeding_turns == 0:
                self._speed_malus = 0
                msg_parts.append(t("pet_combat.bleed_end", lang, pet=self.nickname))

        return "\n".join(msg_parts)

    def apply_special_effect(self, effect: DamageType, lang: str = 'fr') -> str:
        msg = ""
        if effect == DamageType.BITE:
            self._bleeding_turns = 3
            self._speed_malus = self.speed * 0.2
            msg = t("pet_combat.bite_effect", lang)
        elif effect == DamageType.SCRATCH:
            self._acc_malus = self.acc * 0.3
            self._dge_malus = self.dge * 0.3
            msg = t("pet_combat.scratch_effect", lang)
        elif effect == DamageType.POISON:
            self._poisoned_turns = 3
            self._atk_malus = self.atk * 0.3
            msg = t("pet_combat.poison_effect", lang)
        elif effect == DamageType.IMPACT:
            self._stunned_turns = random.randint(1, 2)
            self._defense_malus = self.defense * 0.2
            msg = t("pet_combat.impact_effect", lang)
        elif effect == DamageType.FIRE:
            self._burning_turns = 2
            self._dge_malus = self.dge * 0.8
            msg = t("pet_combat.fire_effect", lang)
        return msg

    @property
    def is_stunned(self) -> bool:
        return self._stunned_turns > 0

    @property
    def real_defense(self):
        return max(0, self.defense - self._defense_malus)

    @property
    def real_acc(self) -> int:
        return max(0, int(self.acc - self._acc_malus))

    @property
    def real_atk(self):
        return max(0, self.atk - self._atk_malus)

    @property
    def real_dge(self):
        return max(0, int(self.dge - self._dge_malus))

    @property
    def real_speed(self):
        return max(1, int(self.speed - self._speed_malus))

    @property
    def thorns_dmg(self) -> float:
        return self.real_defense * 0.1 + (self.real_defense * 0.05 * self._thorn_multiplier)

    def proc_thorns(self) -> float:
        self._thorn_multiplier += 1
        return self.thorns_dmg


    def attack(self, target: 'Pet', can_crit: bool=True, can_effect: bool=True, fatigue_mult: float=1.0, lang: str = 'fr'):
        msgs: list[str] = []
        tick_msg = self.tick_effects(lang)
        if tick_msg:
            msgs.append(tick_msg)

        if not self.is_alive:
            return "\n".join(msgs)

        if self.is_stunned:
            self._stunned_turns -= 1
            msgs.append(t("pet_combat.stunned_log", lang, pet_emoji=self.emoji, pet=self.nickname, turns=self._stunned_turns + 1))
            if self._stunned_turns <= 0:
                self._defense_malus = 0
            return "\n".join(msgs)

        hit_chance = max(20, min(100, int(100 + (self.real_acc * 1.0) - (target.real_dge * fatigue_mult))))
        if random.randint(1, 100) > hit_chance:
            msgs.append(t("pet_combat.dodge_msg", lang, target_emoji=target.emoji, target=target.nickname, attacker=self.nickname))
            return "\n".join(msgs)

        dmg_type = PET_DAMAGE_TYPES.get(self.pet_type)
        is_effect_trigger = can_effect and dmg_type and random.randint(1, 100) < self.spc_c

        min_dmg = self.real_atk * 0.2
        base_dmg = max(min_dmg, self.real_atk - (target.real_defense * fatigue_mult))

        is_crit = random.randint(1, 100) <= self.crit_c if can_crit else False
        crit_mult = random.uniform(1 + (self.crit_d - 1) / 2, self.crit_d) if is_crit else 1.0

        base_dmg = int(base_dmg * crit_mult * random.uniform(0.9, 1.1))
        final_dmg = base_dmg

        new_theorical_hp = max(0, (target.hp - final_dmg))
        hundred_steps = (target.hp - 1) // 100 - new_theorical_hp // 100
        gating_msg = ""
        if hundred_steps > 0:
            tmp_dmg = 0
            final_dmg -= 1
            target.hp -= 1
            for step in range(0, hundred_steps):
                gate_prob = ((target.hp + 1) % 100) / 200 if step < 1 and (target.hp + 1) % 100 != 0 else 0.5
                gate_prob *= 1 - self.real_acc / 100
                if random.random() < gate_prob:  # Gating check sucess
                    tmp_dmg += min(final_dmg, target.hp % 100)
                    final_dmg = 0
                    gating_msg = t("pet_combat.gating", lang, pet=target.nickname, dmg=tmp_dmg)
                    print(f"Gating on {step + 1} step")
                    break
                else:
                    tmp_dmg += min(final_dmg, 100)
                    final_dmg -= min(final_dmg, 100)
            final_dmg = tmp_dmg + final_dmg

        target.hp = max(0, target.hp - final_dmg)
        effect_msg = ""
        if is_effect_trigger and dmg_type is not None:
            effect_msg = target.apply_special_effect(dmg_type, lang)

        if is_crit:
            msg = t("pet_combat.crit_msg", lang, attacker_emoji=self.emoji, attacker=self.nickname, dmg=base_dmg) + effect_msg
        else:
            msg = t("pet_combat.hit_msg", lang, attacker_emoji=self.emoji, attacker=self.nickname, dmg=base_dmg) + effect_msg
        
        if is_effect_trigger and dmg_type == DamageType.SCRATCH:
            heal_amount = int(self.real_atk * 0.3) * (2 if is_crit else 1)
            self.hp = min(self.max_hp, self.hp + heal_amount)
            msg += t("pet_combat.scratch_heal", lang, dmg=heal_amount)


        proba_thorns = min(0.70, target.real_defense / target.real_atk)
        if random.random() < proba_thorns:
            thorns_dmg = int(target.proc_thorns())
            if thorns_dmg > 0:
                self.hp = max(0, self.hp - thorns_dmg)
                msg += t("pet_combat.thorns", lang, dmg=thorns_dmg)

        msgs.append(msg)
        if gating_msg:
            msgs.append(gating_msg)

        return "\n".join(msgs)

    def update_elo(self, opponent: 'Pet', result: float):
        if self.level < 5 or opponent.level < 5:
            return 0, 0
        K = 32
        expected_self = 1 / (1 + 10 ** ((opponent.elo - self.elo) / 400))
        expected_opp = 1 / (1 + 10 ** ((self.elo - opponent.elo) / 400))
        score_opp = 1.0 - result
        diff_self = int(K * (result - expected_self))
        diff_opp = int(K * (score_opp - expected_opp))

        self.elo += diff_self
        opponent.elo += diff_opp

        self.elo = max(0, self.elo)
        opponent.elo = max(0, opponent.elo)

        return diff_self, diff_opp