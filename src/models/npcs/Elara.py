from src.models.NPC import NPC, NPCRegistry
from src.database.npc import get_reputation
from src.utils.i18n import t
from src.items.MysteryEgg import MysteryEgg
from src.items.FarmItem import TomatoSeed, StrawberrySeed, GoldenAppleSeed

@NPCRegistry.register
class ElaraNPC(NPC):
    id = "elara"
    name = "Elara"
    emoji = "🌿"
    color = 0x2e8b57 # Sea Green

    def get_gift_preferences(self):
        return {
            "œuf mystère": 2.0,
            "fruit étoile": 2.0,
            "pomme dorée": 2.0,
            "graine de tomate": 1.5,
            "graine de citrouille": 1.5,
            "fraise": 1.5,
            "œuf saison": 1.5,
            "graine de cacao": 1.5,
            "graine de café": 1.5,
            "graine de fraise": 1.5,
            "pépin de pomme dorée": 1.5,
            "pépin de fruit étoile": 1.5,
            "aimant rouillé": 0.5
        }

    def get_bonus(self, user_id: int):
        rep_data = get_reputation(user_id, self.id)
        lvl = rep_data["level"]
        # Reduce grow time by level * 2% (max 20%)
        return {"farming_speed_boost": min(lvl * 0.02, 0.20)}

    def get_shop_inventory(self, current_level: int) -> list:
        items = []
        if current_level >= 1:
            items.append(TomatoSeed())
        if current_level >= 2:
            items.append(MysteryEgg())
        if current_level >= 3:
            items.append(StrawberrySeed())
        if current_level >= 4:
            items.append(GoldenAppleSeed())
        return items

    def get_greeting(self, user_id: int, lang: str):
        rep_data = get_reputation(user_id, self.id)
        lvl = rep_data["level"]
        if lvl >= 5:
            return t("npcs.elara.greeting_high", lang, default="Bonjour mon cher ! La nature chante en ta présence.")
        elif lvl >= 2:
            return t("npcs.elara.greeting_med", lang, default="Ravi de te revoir. Mes plantes poussent merveilleusement bien aujourd'hui.")
        else:
            return t("npcs.elara.greeting_low", lang, default="Bonjour. Prends soin de la terre et des familiers, s'il te plaît.")
