from src.models.npcs.GambleBot import GambleBotNPC

# All NPC classes should be imported here to ensure they are registered in NPCRegistry
__all__ = ["GambleBotNPC"]

def initialize_npcs():
    # Registry is populated automatically via the @NPCRegistry.register decorator
    # when the classes are imported above.
    print("✅ NPCs initialisés dans le système.")
