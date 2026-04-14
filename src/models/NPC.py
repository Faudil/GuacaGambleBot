from typing import List, Dict, Any, Optional
from src.utils.i18n import t

class NPC:
    id: str = ""
    name: str = ""
    emoji: str = ""
    color: int = 0x000000

    def __init__(self):
        self.gift_preferences: Dict[str, float] = self.get_gift_preferences()

    def get_gift_preferences(self) -> Dict[str, float]:
        """Override to define item multipliers for gifts."""
        return {}

    def get_greeting(self, user_id: int, lang: str) -> str:
        """Returns a greeting based on reputation level."""
        from src.database.npc import get_reputation
        rep_data = get_reputation(user_id, self.id)
        lvl = rep_data["level"]
        return t(f"npcs.{self.id}.greeting_lvl{lvl}", lang, default=t(f"npcs.{self.id}.greeting_default", lang))

    def get_daily_rep_cap(self) -> int:
        """Returns the flat daily cap for reputation points."""
        return 500

    def get_rank_name(self, level: int, lang: str) -> str:
        """Override to return custom rank names."""
        names = ["Inconnu", "Connaissance", "Associé", "Ami", "Partenaire"]
        idx = min(level - 1, len(names) - 1)
        name = names[idx]
        return name

    def get_rank_up_cost(self, current_level: int) -> list:
        """Override to provide items needed for the next rank. 
           Format: [{'name': 'item_name', 'quantity': 1}]
        """
        return []

    def get_shop_inventory(self, current_level: int) -> list:
        """Override to provide list of uninstantiated Item classes this NPC sells."""
        return []

    def get_bonus(self, user_id: int) -> Dict[str, Any]:
        return {}

    def on_gift(self, user_id: int, item_name: str, quantity: int) -> int:
        base_points = 10 * quantity
        multiplier = self.gift_preferences.get(item_name.lower(), 1.0)
        return int(base_points * multiplier)

class NPCRegistry:
    _npcs: Dict[str, NPC] = {}

    @classmethod
    def register(cls, npc_class: type):
        npc_instance = npc_class()
        cls._npcs[npc_instance.id] = npc_instance
        return npc_class

    @classmethod
    def get_npc(cls, npc_id: str) -> Optional[NPC]:
        return cls._npcs.get(npc_id)

    @classmethod
    def get_all_npcs(cls) -> List[NPC]:
        return list(cls._npcs.values())
