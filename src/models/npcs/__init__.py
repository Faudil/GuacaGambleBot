from src.models.npcs.GambleBot import GambleBotNPC
from src.models.npcs.Thorek import ThorekNPC
from src.models.npcs.Elara import ElaraNPC
from src.models.npcs.Irian import IrianNPC

# All NPC classes should be imported here to ensure they are registered in NPCRegistry
__all__ = ["GambleBotNPC", "ThorekNPC", "ElaraNPC", "IrianNPC"]

def initialize_npcs():
    # Registry is populated automatically via the @NPCRegistry.register decorator
    # when the classes are imported above.
    print("✅ NPCs initialisés dans le système.")
