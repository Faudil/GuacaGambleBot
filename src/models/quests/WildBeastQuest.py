from typing import List, Dict, Any
from src.models.Quest import Quest, QuestStep, QuestStepType, QuestReward, QuestType, QuestRegistry

class WildBeastQuest(Quest):
    id = "wild_beast"
    type = QuestType.MAIN

    def get_steps(self) -> List[QuestStep]:
        return [
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.wild_beast.step0_irian",
                rewards=QuestReward(xp=50) # Just dialogue
            ),
            QuestStep(
                step_type=QuestStepType.BOSS_BATTLE,
                text_key="quests.wild_beast.step1_boss",
                rewards=QuestReward(items=["œuf mystère"], xp=200),
                boss_stats={
                    "name": "Loup Enragé",
                    "emoji": "🐺",
                    "level": 3,
                    "hp": 40,
                    "atk": 12,
                    "def": 5,
                    "spd": 15,
                    "dge": 5,
                    "acc": 10,
                    "crit_c": 5,
                    "crit_d": 1.5
                }
            ),
            QuestStep(
                step_type=QuestStepType.ACTIVITY,
                text_key="quests.wild_beast.step2_hatch",
                rewards=QuestReward(money=300, xp=200),
                target_stat="pets_hatched",
                target_count=1
            )
        ]

    def on_activity(self, user_id: int, activity: str, amount: int, current_progress: Dict[str, Any]) -> bool:
        step_idx = current_progress.get('step_index', 0)
        if step_idx != 2:
            return False
            
        step = self.steps[step_idx]
        target_stat = step.extra.get('target_stat')
        if activity != target_stat:
            return False
            
        new_val = current_progress.get('progress_value', 0) + amount
        from src.database.quest import update_quest_progress
        update_quest_progress(user_id, self.id, progress_value=new_val)
        
        return new_val >= step.extra.get('target_count', 1)

    def get_unlocks(self) -> List[str]:
        return ["building_scraps"]

# Register the quest
QuestRegistry.register(WildBeastQuest)
