from typing import List, Dict, Any
from src.models.Quest import Quest, QuestStep, QuestStepType, QuestReward, QuestType, QuestRegistry

class ScrapFarmQuest(Quest):
    id = "scrap_farm"
    type = QuestType.MAIN

    def get_steps(self) -> List[QuestStep]:
        return [
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.scrap_farm.step0_elara",
                rewards=QuestReward(money=1000)
            ),
            QuestStep(
                step_type=QuestStepType.ACTIVITY,
                text_key="quests.scrap_farm.step1_buy_plot",
                rewards=QuestReward(items=["wheat_seed", "wheat_seed"]),
                target_stat="farm_plots_bought",
                target_count=1
            ),
            QuestStep(
                step_type=QuestStepType.ACTIVITY,
                text_key="quests.scrap_farm.step2_plant",
                rewards=QuestReward(xp=150),
                target_stat="crops_planted",
                target_count=1
            )
        ]

    def on_activity(self, user_id: int, activity: str, amount: int, current_progress: Dict[str, Any]) -> bool:
        step_idx = current_progress.get('step_index', 0)
        if step_idx not in [1, 2]:
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
        return ["wild_beast"]

# Register the quest
QuestRegistry.register(ScrapFarmQuest)
