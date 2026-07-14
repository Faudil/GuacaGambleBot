from typing import Dict, Any
from src.models.NPC import NPCRegistry

class NPCManager:
    @staticmethod
    def get_user_bonuses(user_id: int) -> Dict[str, Any]:
        """Aggregates all passive bonuses from NPCs the user has reputation with."""
        total_bonuses = {
            "shop_discount": 0.0,
            "gamble_payout": 0.0,
            "xp_boost": 0.0,
            "mining_risk_reduction": 0,
            "farming_speed_boost": 0.0,
            "fishing_time_bonus": 0.0
        }
        
        npcs = NPCRegistry.get_all_npcs()
        for npc in npcs:
            npc_bonuses = npc.get_bonus(user_id)
            for key, value in npc_bonuses.items():
                if key in total_bonuses:
                    if key == "shop_discount":
                        total_bonuses[key] = max(total_bonuses[key], value) # Deepest discount wins
                    elif key == "farming_speed_boost":
                        total_bonuses[key] = max(total_bonuses[key], value) # Deepest speed boost wins
                    else:
                        total_bonuses[key] += value
                        
        return total_bonuses
